"""Execute release orchestration against a fake Kubernetes command, including failures."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class ReleaseGateTests(unittest.TestCase):
    def run_release(self, failed):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            log = root / 'calls'
            kubectl = root / 'kubectl'
            kubectl.write_text('''#!/usr/bin/env python3
import base64, os, sys
args = sys.argv[1:]
with open(os.environ['CALL_LOG'], 'a') as f: f.write(' '.join(args)+'\\n')
if args[:2] == ['get', 'secret']:
    print('gym-jinni-auth\\n'+base64.b64encode(b'x'*48).decode())
elif args[0] == 'apply' and 'migration.yml' in args[2]:
    print('job.batch/gym-jinni-migrate-123')
elif args[:2] == ['get', 'job.batch/gym-jinni-migrate-123']:
    print('/True' if os.environ['FAILED'] == '1' else 'True/')
''')
            kubectl.chmod(0o755)
            env = {**os.environ, 'PATH': directory + os.pathsep + os.environ['PATH'],
                   'CALL_LOG': str(log), 'FAILED': str(int(failed))}
            env.pop('JWT_SECRET', None)
            env.pop('KUBE_NAMESPACE', None)
            result = subprocess.run(['bash', str(ROOT / 'k8s-setup/scripts/deploy-release.sh'), directory],
                                    env=env, capture_output=True, text=True)
            return result, log.read_text()

    def test_failed_migration_never_applies_application(self):
        result, calls = self.run_release(True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('application rollout was blocked', result.stderr)
        self.assertNotIn('application.yml', calls)
        self.assertNotIn('rollout status deployment/', calls)

    def test_successful_migration_precedes_application_rollout(self):
        result, calls = self.run_release(False)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertLess(calls.index('get job.batch/'), calls.index('application.yml'))
        self.assertIn('rollout status deployment/main-service', calls)
