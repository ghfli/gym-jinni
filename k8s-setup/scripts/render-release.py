#!/usr/bin/env python3
"""Render validated release artifacts without contacting Kubernetes.

Remote credentials must already exist in postgres-secret, including DBURL.
The output intentionally contains no remote database credentials.
"""
import argparse
import copy
import hashlib
import re
import uuid
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[1]


def load(name):
    return list(yaml.safe_load_all((ROOT / 'manifests' / name).read_text()))


def render(profile, domain, service_image=None, ui_image=None, pull_secret=None):
    if not re.fullmatch(r'(?=.{1,253}$)[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?', domain) or any(
        not re.fullmatch(r'[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?', label)
        for label in domain.split('.')
    ):
        raise ValueError('domain must be a DNS hostname without a scheme or port')
    if profile not in ('local', 'remote'):
        raise ValueError('unknown deployment profile')
    if profile == 'remote':
        for image in (service_image, ui_image):
            if not image or not re.fullmatch(r'[a-z0-9][a-z0-9./_-]*(?::[0-9]+)?[a-z0-9./_-]*(?::[a-f0-9]{7,64}|@sha256:[a-f0-9]{64})', image):
                raise ValueError('remote images require an immutable commit tag or sha256 digest')
    service_image = service_image or 'gym-jinni/service:latest'
    ui_image = ui_image or 'gym-jinni/ui:latest'
    if pull_secret and not re.fullmatch(r'[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?', pull_secret):
        raise ValueError('invalid registry pull secret name')
    backend = load('main-service-deployment.yml')
    ui = load('ui-deployment.yml')
    config = load('configmaps.yml')
    pod = next(x for x in backend if x['kind'] == 'Deployment')['spec']['template']['spec']
    migration = copy.deepcopy(next(x for x in pod['initContainers'] if x['name'] == 'run-migrations'))
    pod['initContainers'] = [x for x in pod['initContainers'] if x['name'] != 'run-migrations']
    ui_pod = next(x for x in ui if x['kind'] == 'Deployment')['spec']['template']['spec']
    for container, image in ((pod['containers'][0], service_image), (migration, service_image),
                             (ui_pod['containers'][0], ui_image)):
        container['image'] = image
        container['imagePullPolicy'] = 'Never' if profile == 'local' else 'IfNotPresent'
    if profile == 'local':
        # Rebuilt imported tags need a new pod template even when the tag is unchanged.
        for workload in (backend, ui):
            deployment = next(x for x in workload if x['kind'] == 'Deployment')
            deployment['spec']['template']['metadata']['annotations'] = {'gym-jinni/local-build': uuid.uuid4().hex}
    # Credentials are provisioned separately for a remote deployment.
    if profile == 'remote':
        config[0]['data'].pop('DBURL', None)
        for container in (pod['containers'][0], migration):
            next(x for x in container['env'] if x['name'] == 'DBURL')['valueFrom'] = {
                'secretKeyRef': {'name': 'postgres-secret', 'key': 'DBURL', 'optional': False}}
    job_pod = {'restartPolicy': 'Never', 'initContainers': copy.deepcopy(pod['initContainers']),
               'containers': [migration]}
    if pull_secret:
        for spec in (pod, ui_pod, job_pod):
            spec['imagePullSecrets'] = [{'name': pull_secret}]
    release_hash = hashlib.sha256(service_image.encode())
    for path in sorted((ROOT.parent / 'service/mig').glob('*.sql')):
        release_hash.update(path.name.encode())
        release_hash.update(path.read_bytes())
    release_id = release_hash.hexdigest()[:16]
    job = {'apiVersion': 'batch/v1', 'kind': 'Job',
           'metadata': {'name': f'gym-jinni-migrate-{release_id}', 'namespace': 'gym-jinni'},
           'spec': {'backoffLimit': 0, 'activeDeadlineSeconds': 600, 'template': {'spec': job_pod}}}
    ingress = load('ingress.yml')
    ingress[0]['spec']['tls'][0]['hosts'] = [domain]
    ingress[0]['spec']['rules'][0]['host'] = domain
    # Gateway paths are /v1/*; neither rewrite nor the former /api/v1 prefix is valid.
    ingress[0]['metadata']['annotations'].pop('nginx.ingress.kubernetes.io/rewrite-target', None)
    ingress[0]['spec']['rules'][0]['http']['paths'][1]['path'] = '/v1'
    if profile == 'local':
        ingress[0]['spec'].pop('tls')
        ingress[0]['metadata']['annotations'] = {}
    result = {'foundation.yml': config + load('postgres-statefulset.yml'),
              'migration.yml': [job], 'application.yml': backend + ui + ingress}
    if profile == 'local':
        result['foundation.yml'] = load('secrets.yml') + result['foundation.yml']
    for docs in result.values():
        text = yaml.safe_dump_all(docs)
        if re.search(r'\$\{[A-Z_][A-Z_0-9]*\}', text.replace('${DBURL}', '')):
            raise ValueError('unresolved manifest placeholder')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--profile', choices=['local', 'remote'], required=True)
    parser.add_argument('--domain', required=True)
    parser.add_argument('--service-image')
    parser.add_argument('--ui-image')
    parser.add_argument('--pull-secret')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    try:
        result = render(args.profile, args.domain, args.service_image, args.ui_image, args.pull_secret)
    except ValueError as exc:
        parser.error(str(exc))
    args.output.mkdir(parents=True, exist_ok=True)
    for name, docs in result.items():
        (args.output / name).write_text(yaml.safe_dump_all(docs, sort_keys=False))


if __name__ == '__main__':
    main()
