#!/usr/bin/env python3
"""Opt-in real rollout acceptance. Creates/deletes only a uniquely named k3d cluster.

Build current service/UI images first. No active kubeconfig is read or changed.
"""
import argparse
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid

import yaml

ROOT = Path(__file__).resolve().parents[2]
SCRIPTS = ROOT / 'k8s-setup/scripts'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--service-image', default='gym-jinni/service:mvp')
    parser.add_argument('--ui-image', default='gym-jinni/ui:latest')
    parser.add_argument('--k3s-image', default='rancher/k3s:v1.31.5-k3s1')
    args = parser.parse_args()
    for tool in ('k3d', 'kubectl', 'docker'):
        if not shutil.which(tool):
            parser.error(f'{tool} is required')
    prefix = []
    if subprocess.run(['docker', 'info'], capture_output=True).returncode:
        prefix = ['sudo', '-n']
    name = 'gj-acceptance-' + uuid.uuid4().hex[:10]
    with tempfile.TemporaryDirectory(prefix=name) as directory:
        root = Path(directory)
        env = {**os.environ, 'KUBECONFIG': str(root / 'kubeconfig')}
        env.pop('JWT_SECRET', None)
        env.pop('KUBE_NAMESPACE', None)
        os.chmod(root, 0o700)

        def run(command, *, check=True, timeout=360):
            result = subprocess.run(command, env=env, text=True, capture_output=True, timeout=timeout)
            if check and result.returncode:
                raise RuntimeError(f'{command[0]} {command[1]} failed: {result.stderr[-2000:]}')
            return result

        def kube(*command, check=True):
            return run(['kubectl', *command], check=check)

        with socket.socket() as api_socket:
            api_socket.bind(('127.0.0.1', 0))
            api_port = api_socket.getsockname()[1]
        forward = None
        try:
            run(prefix + ['k3d', 'cluster', 'create', name, '--image', args.k3s_image,
                          '--api-port', f'127.0.0.1:{api_port}', '--kubeconfig-switch-context=false',
                          '--kubeconfig-update-default=false', '--timeout', '180s'])
            config = run(prefix + ['k3d', 'kubeconfig', 'get', name]).stdout
            with open(root / 'kubeconfig', 'x', opener=lambda p, flags: os.open(p, flags, 0o600)) as stream:
                stream.write(config)
            run(prefix + ['k3d', 'image', 'import', args.service_image, args.ui_image,
                          'postgres:15-alpine', '-c', name])
            run(['python3', str(SCRIPTS / 'render-release.py'), '--profile', 'local',
                 '--domain', 'localhost', '--service-image', args.service_image,
                 '--ui-image', args.ui_image, '--output', str(root)])
            kube('apply', '-f', str(ROOT / 'k8s-setup/manifests/namespace.yml'))
            run(['bash', str(SCRIPTS / 'deploy-release.sh'), str(root)], timeout=1000)
            initial = json.loads(kube('get', 'secret', 'gym-jinni-auth', '-n', 'gym-jinni', '-o', 'json').stdout)['data']['JWT_SECRET']
            run(['bash', str(SCRIPTS / 'deploy-release.sh'), str(root)], timeout=1000)
            repeated = json.loads(kube('get', 'secret', 'gym-jinni-auth', '-n', 'gym-jinni', '-o', 'json').stdout)['data']['JWT_SECRET']
            if initial != repeated:
                raise RuntimeError('Reapply changed the signing key')
            print('PASS real startup, migration Job and stable signing key on reapply', flush=True)

            kube('apply', '-f', str(ROOT / 'k8s-setup/manifests/deployer-rbac.yml'))
            account = '--as=system:serviceaccount:gym-jinni:gym-jinni-deployer'
            for resource, namespace, expected in [('jobs', 'gym-jinni', True), ('secrets', 'gym-jinni', True),
                                                   ('secrets', 'default', False), ('clusterissuers', None, False)]:
                command = ['auth', 'can-i', 'create', resource, account]
                if namespace:
                    command += ['-n', namespace]
                allowed = kube(*command, check=False).returncode == 0
                if allowed != expected:
                    raise RuntimeError(f'Unexpected deployer permission for {resource} in {namespace}')
            print('PASS real namespace-scoped deployer allow/deny checks', flush=True)

            with socket.socket() as sock:
                sock.bind(('127.0.0.1', 0))
                port = sock.getsockname()[1]
            forward = subprocess.Popen(['kubectl', 'port-forward', '-n', 'gym-jinni', 'service/main-service',
                                        f'{port}:8081', '--address=127.0.0.1'], env=env,
                                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            def request(path, payload=None, token=None):
                headers = {'Content-Type':'application/json'}
                if token:
                    headers['Authorization'] = 'Bearer ' + token
                req = urllib.request.Request(f'http://127.0.0.1:{port}' + path,
                                              data=json.dumps(payload).encode() if payload is not None else None,
                                              headers=headers)
                try:
                    with urllib.request.urlopen(req, timeout=10) as response:
                        return response.status, json.load(response)
                except urllib.error.HTTPError as error:
                    return error.code, {}
            for _ in range(60):
                try:
                    request('/v1/roles/by-name/customer')
                    break
                except (OSError, urllib.error.URLError):
                    time.sleep(1)
            credentials = {'user':{'name':'Acceptance', 'email':name+'@example.com',
                                   'phone':'1234567890', 'passwd':'acceptance-password'}}
            if request('/v1/create_user', credentials)[0] != 200:
                raise RuntimeError('Registration failed')
            code, login = request('/v1/login_user', credentials)
            if code != 200:
                raise RuntimeError('Login failed')
            key_file = root / 'new-key'
            key_file.write_text(secrets.token_urlsafe(48))
            key_file.chmod(0o600)
            run(['python3', str(SCRIPTS / 'rotate-jwt-secret.py'), '--key-file', str(key_file)])
            # Port-forward pins a pod; replace it after the rotation rollout.
            forward.terminate(); forward.wait(timeout=10)
            forward = subprocess.Popen(['kubectl', 'port-forward', '-n', 'gym-jinni', 'service/main-service',
                                        f'{port}:8081', '--address=127.0.0.1'], env=env,
                                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            time.sleep(2)
            token = login.get('accessTkn', login.get('access_tkn'))
            if request('/v1/classes', token=token)[0] != 401 or request('/v1/login_user', credentials)[0] != 200:
                raise RuntimeError('Signing-key rotation acceptance failed')
            print('PASS old tokens rejected and new login succeeds after key rotation', flush=True)

            migration_file = root / 'migration.yml'
            job = yaml.safe_load(migration_file.read_text())
            job['metadata']['name'] += '-failure'
            next(c for c in job['spec']['template']['spec']['initContainers'] if c['name'] == 'run-migrations')['args'] = ['exit 17']
            migration_file.write_text(yaml.safe_dump(job))
            applications = list(yaml.safe_load_all((root / 'application.yml').read_text()))
            deployment = next(d for d in applications if d['kind'] == 'Deployment' and d['metadata']['name'] == 'main-service')
            deployment['spec']['replicas'] = 0
            (root / 'application.yml').write_text(yaml.safe_dump_all(applications))
            result = run(['bash', str(SCRIPTS / 'deploy-release.sh'), str(root)], check=False, timeout=1000)
            live = json.loads(kube('get', 'deployment', 'main-service', '-n', 'gym-jinni', '-o', 'json').stdout)
            if result.returncode == 0 or live['spec']['replicas'] != 2:
                raise RuntimeError('Failed migration did not block application change')
            print('PASS failed real migration Job blocks application rollout', flush=True)
        finally:
            if forward:
                forward.terminate()
                forward.wait(timeout=10)
            run(prefix + ['k3d', 'cluster', 'delete', name], check=False, timeout=180)


if __name__ == '__main__':
    main()
