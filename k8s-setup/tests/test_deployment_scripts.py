"""Exercise deployment guards/readiness without contacting a Kubernetes cluster."""
import base64
import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile
import unittest

SCRIPTS = Path(__file__).resolve().parents[1] / "scripts"
FAKE_KUBECTL = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
args = sys.argv[1:]
with open(os.environ['KUBECTL_LOG'], 'a') as log:
    log.write(json.dumps(args) + '\n')
mode = os.environ.get('FAKE_MODE', '')
if args[:2] == ['config', 'current-context']:
    sys.exit(1 if mode == 'no-context' else 0)
if args[:2] == ['config', 'view']:
    print('' if mode == 'no-server' else 'https://cluster.example.test:6443')
    sys.exit(0)
if args[0].startswith('--request-timeout='):
    sys.exit(1 if mode == 'unreachable' else 0)
if args[:2] == ['rollout', 'status']:
    sys.exit(1 if mode == 'rollout-failure' else 0)
if args[0] == 'wait':
    sys.exit(1 if mode == 'crd-failure' else 0)
if '--dry-run=server' in args:
    manifest = sys.stdin.read()
    assert 'kind: Issuer' in manifest and 'selfSigned: {}' in manifest
    counter = pathlib.Path(os.environ['CHECK_COUNTER'])
    attempt = int(counter.read_text()) + 1 if counter.exists() else 1
    counter.write_text(str(attempt))
    sys.exit(1 if mode == 'webhook-failure' or (mode == 'webhook-delayed' and attempt < 3) else 0)
sys.exit(0)
'''


class DeploymentScriptsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.log = self.root / "kubectl.log"
        self.counter = self.root / "counter"
        self.config = self.root / "config"
        for name, source in {"kubectl": FAKE_KUBECTL, "sleep": "#!/bin/sh\nexit 0\n"}.items():
            executable = self.root / name
            executable.write_text(source)
            executable.chmod(0o700)
        self.env = dict(os.environ, PATH=f"{self.root}:{os.environ['PATH']}",
                        KUBECTL_LOG=str(self.log), CHECK_COUNTER=str(self.counter),
                        KUBECONFIG=str(self.config))
        self.env.pop("KUBECONFIG_BASE64", None)
        self.env.pop("FAKE_MODE", None)

    def run_script(self, name, mode="", encoded=None):
        self.env["FAKE_MODE"] = mode
        if encoded is not None:
            self.env["KUBECONFIG_BASE64"] = encoded
        return subprocess.run(["bash", str(SCRIPTS / name)], env=self.env,
                              text=True, capture_output=True, timeout=10)

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []

    def test_missing_credentials_fail_before_kubectl(self):
        result = self.run_script("configure-kubectl.sh")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("secret is missing", result.stderr)
        self.assertFalse(self.config.exists())
        self.assertEqual(self.calls(), [])

    def test_invalid_base64_is_rejected_without_logging_secret(self):
        value = "not base64!"
        result = self.run_script("configure-kubectl.sh", encoded=value)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("not valid base64", result.stderr)
        self.assertNotIn(value, result.stdout + result.stderr)
        self.assertFalse(self.config.exists())
        self.assertEqual(self.calls(), [])

    def test_empty_decoded_file_is_rejected(self):
        result = self.run_script("configure-kubectl.sh", encoded="\n")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("current context", result.stderr)
        self.assertEqual(self.calls(), [])

    def test_invalid_context_endpoint_and_unreachable_api(self):
        encoded = base64.b64encode(b"opaque-fixture-credentials").decode()
        for mode, message in [("no-context", "current context"),
                              ("no-server", "API server"),
                              ("unreachable", "Cannot reach or authenticate")]:
            with self.subTest(mode=mode):
                result = self.run_script("configure-kubectl.sh", mode, encoded)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(message, result.stderr)
                self.assertNotIn("opaque-fixture-credentials", result.stdout + result.stderr)

    def test_valid_config_is_private_and_api_is_checked(self):
        payload = b"opaque-fixture-credentials"
        result = self.run_script("configure-kubectl.sh", encoded=base64.b64encode(payload).decode())
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.config.read_bytes(), payload)
        self.assertEqual(stat.S_IMODE(self.config.stat().st_mode), 0o600)
        self.assertIn(["--request-timeout=15s", "get", "--raw=/readyz"], self.calls())
        self.assertNotIn(payload.decode(), result.stdout + result.stderr)

    def test_waits_for_all_components_and_webhook_admission(self):
        result = self.run_script("install-cert-manager.sh", "webhook-delayed")
        self.assertEqual(result.returncode, 0, result.stderr)
        rollouts = [call[2] for call in self.calls() if call[:2] == ["rollout", "status"]]
        self.assertEqual(rollouts, ["deployment/cert-manager", "deployment/cert-manager-cainjector", "deployment/cert-manager-webhook"])
        self.assertEqual(self.counter.read_text(), "3")
        self.assertIn("webhook are ready", result.stdout)

    def test_failed_rollout_stops_before_admission(self):
        result = self.run_script("install-cert-manager.sh", "rollout-failure")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.counter.exists())

    def test_failed_crd_registration_stops_before_rollouts(self):
        result = self.run_script("install-cert-manager.sh", "crd-failure")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(call[:2] == ["rollout", "status"] for call in self.calls()))

    def test_webhook_failure_has_bounded_retries_and_diagnostics(self):
        result = self.run_script("install-cert-manager.sh", "webhook-failure")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.counter.read_text(), "30")
        self.assertIn("did not become ready", result.stderr)
        self.assertIn(["get", "deployments,pods,services", "-n", "cert-manager"], self.calls())


if __name__ == "__main__":
    unittest.main()
