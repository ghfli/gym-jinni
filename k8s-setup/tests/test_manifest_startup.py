"""Validate real manifest init commands and signing-secret lifecycle without a cluster."""
import base64
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

import yaml
from manifest_utils import named_container

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "k8s-setup/scripts/ensure-jwt-secret.sh"
MANIFEST = ROOT / "k8s-setup/manifests/main-service-deployment.yml"
FAKE_KUBECTL = r'''#!/usr/bin/env python3
import base64, json, os, pathlib, stat, sys
args = sys.argv[1:]
store = pathlib.Path(os.environ['SECRET_STORE'])
mode = os.environ.get('FAKE_MODE', '')
with open(os.environ['COMMAND_LOG'], 'a') as output:
    output.write(json.dumps(args) + '\n')
if args[:2] == ['get', 'secret']:
    if mode == 'get-denied': sys.exit(1)
    if store.exists():
        data = json.loads(store.read_text())
        if 'json' in args:
            print(json.dumps({'metadata': {'name': 'gym-jinni-auth', 'resourceVersion': '1'}, 'data': data}))
        else:
            print('gym-jinni-auth\n' + data.get('JWT_SECRET', ''))
    sys.exit(0)
if args[:2] == ['get', 'deployment']:
    sys.exit(1 if mode == 'deployment-missing' else 0)
if args[:2] == ['patch', 'secret']:
    if mode == 'patch-conflict': sys.exit(1)
    filename = args[args.index('--patch-file') + 1]
    assert stat.S_IMODE(pathlib.Path(filename).stat().st_mode) == 0o600
    patch = json.loads(pathlib.Path(filename).read_text())
    assert patch[0] == {'op': 'test', 'path': '/metadata/resourceVersion', 'value': '1'}
    data = json.loads(store.read_text())
    data['JWT_SECRET'] = patch[1]['value']
    store.write_text(json.dumps(data))
    sys.exit(0)
if args[:2] == ['rollout', 'restart'] or args[:2] == ['rollout', 'status']:
    sys.exit(1 if mode == 'rollout-failure' else 0)
if args[:3] == ['create', 'secret', 'generic']:
    if mode == 'create-denied': sys.exit(1)
    if mode == 'race':
        store.write_text(json.dumps({'JWT_SECRET': base64.b64encode(b'race-winner-signing-key-at-least-32-bytes').decode()}))
        sys.exit(1)
    if store.exists(): sys.exit(1)
    filename = next(a.split('=', 2)[2] for a in args if a.startswith('--from-file=JWT_SECRET='))
    assert stat.S_IMODE(pathlib.Path(filename).stat().st_mode) == 0o600
    payload = pathlib.Path(filename).read_bytes()
    store.write_text(json.dumps({'JWT_SECRET': base64.b64encode(payload).decode()}))
    sys.exit(0)
sys.exit(2)
'''


class StartupContractTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.store = self.root / "secret.json"
        self.log = self.root / "commands.jsonl"
        self.env = dict(os.environ, PATH=f"{self.root}:{os.environ['PATH']}",
                        SECRET_STORE=str(self.store), COMMAND_LOG=str(self.log),
                        TMPDIR=str(self.root))
        self.env.pop("JWT_SECRET", None)
        self.env.pop("FAKE_MODE", None)
        self.env.pop("KUBE_NAMESPACE", None)
        executable = self.root / "kubectl"
        executable.write_text(FAKE_KUBECTL)
        executable.chmod(0o700)

    def run_helper(self, key=None, mode=""):
        self.env["FAKE_MODE"] = mode
        if key is not None:
            self.env["JWT_SECRET"] = key
        return subprocess.run(["bash", str(SCRIPT)], env=self.env, text=True,
                              capture_output=True, timeout=10)

    def stored(self):
        return base64.b64decode(json.loads(self.store.read_text())["JWT_SECRET"]).decode()

    def seed(self, key):
        self.store.write_text(json.dumps({"JWT_SECRET": base64.b64encode(key.encode()).decode()}))

    def test_generated_key_is_valid_private_and_stable_on_redeploy(self):
        first = self.run_helper()
        self.assertEqual(first.returncode, 0, first.stderr)
        key = self.stored()
        self.assertGreaterEqual(len(key.encode()), 32)
        self.assertNotIn(key, first.stdout + first.stderr + self.log.read_text())
        second = self.run_helper()
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertEqual(self.stored(), key)
        calls = [json.loads(line) for line in self.log.read_text().splitlines()]
        self.assertEqual(sum(call[:2] == ["create", "secret"] for call in calls), 1)
        self.assertFalse([path for path in self.root.iterdir() if path.is_dir()])

    def test_explicit_key_is_preserved_without_logging(self):
        key = "operator-provided-private-signing-key-with-32-bytes"
        result = self.run_helper(key)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.stored(), key)
        self.assertNotIn(key, result.stdout + result.stderr + self.log.read_text())
        self.assertEqual(self.run_helper(key).returncode, 0)

    def test_invalid_explicit_keys_fail_before_cluster_access(self):
        for key in ["short", "gym-jinni-dev-secret-change-in-prod"]:
            with self.subTest(key=key):
                result = self.run_helper(key)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.store.exists())
                self.assertFalse(self.log.exists())
                self.assertNotIn(key, result.stdout + result.stderr)

    def test_invalid_existing_secret_fails_without_replacement(self):
        self.seed("short")
        before = self.store.read_bytes()
        result = self.run_helper()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.store.read_bytes(), before)
        self.assertIn("valid existing", result.stderr)

    def test_missing_existing_key_fails_without_replacement(self):
        self.store.write_text("{}")
        self.assertNotEqual(self.run_helper().returncode, 0)
        self.assertEqual(self.store.read_text(), "{}")

    def test_different_explicit_key_does_not_rotate_existing_secret(self):
        self.seed("original-signing-key-that-is-at-least-32-bytes")
        before = self.store.read_bytes()
        result = self.run_helper("replacement-signing-key-that-is-at-least-32-bytes")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.store.read_bytes(), before)
        self.assertIn("rotate-jwt-secret.py", result.stderr)

    def test_newline_bytes_are_preserved_in_creation_and_comparison(self):
        key = "operator-signing-key-at-least-32-bytes" + "\n\n"
        self.assertEqual(self.run_helper(key).returncode, 0)
        self.assertEqual(self.stored(), key)
        self.assertEqual(self.run_helper(key).returncode, 0)
        self.assertNotEqual(self.run_helper(key.rstrip("\n")).returncode, 0)
        self.assertEqual(self.stored(), key)

    def test_short_key_cannot_pass_due_to_trimmed_newlines(self):
        self.seed("short\n")
        self.assertNotEqual(self.run_helper().returncode, 0)

    def test_namespace_mismatch_fails_before_access(self):
        self.env["KUBE_NAMESPACE"] = "another-namespace"
        result = self.run_helper()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("match the checked-in manifests", result.stderr)
        self.assertFalse(self.log.exists())

    def test_rotation_rejects_unsafe_files_and_namespace_before_access(self):
        tool = ROOT / "k8s-setup/scripts/rotate-jwt-secret.py"
        key_file = self.root / "replacement"
        key_file.write_text("replacement-signing-key-at-least-32-bytes")
        key_file.chmod(0o644)
        result = subprocess.run(["python3", str(tool), "--key-file", str(key_file)], env=self.env,
                                text=True, capture_output=True, timeout=10)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())
        key_file.chmod(0o600)
        key_file.write_bytes(b"replacement-signing-key-at-least-32-bytes\0")
        result = subprocess.run(["python3", str(tool), "--key-file", str(key_file)], env=self.env,
                                text=True, capture_output=True, timeout=10)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())
        key_file.write_text("replacement-signing-key-at-least-32-bytes")
        self.env["KUBE_NAMESPACE"] = "wrong-namespace"
        result = subprocess.run(["python3", str(tool), "--key-file", str(key_file)], env=self.env,
                                text=True, capture_output=True, timeout=10)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())

    def test_rotation_uses_cas_preserves_other_data_and_restarts(self):
        key = "replacement-signing-key-at-least-32-bytes\n"
        self.seed("original-signing-key-that-is-at-least-32-bytes")
        data = json.loads(self.store.read_text())
        data["other-key"] = "dW5jaGFuZ2Vk"
        self.store.write_text(json.dumps(data))
        key_file = self.root / "replacement"
        key_file.write_text(key)
        key_file.chmod(0o600)
        tool = ROOT / "k8s-setup/scripts/rotate-jwt-secret.py"
        for _ in range(2):
            result = subprocess.run(["python3", str(tool), "--key-file", str(key_file)],
                                    env=self.env, text=True, capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertNotIn(key, result.stdout + result.stderr)
        self.assertEqual(self.stored(), key)
        self.assertEqual(json.loads(self.store.read_text())["other-key"], "dW5jaGFuZ2Vk")
        calls = [json.loads(line) for line in self.log.read_text().splitlines()]
        self.assertEqual(sum(call[:2] == ["patch", "secret"] for call in calls), 1)
        self.assertEqual(sum(call[:2] == ["rollout", "restart"] for call in calls), 2)
        self.assertFalse(any(call[0] == "delete" for call in calls))

    def test_rotation_conflict_and_failed_rollout_are_visible_and_retryable(self):
        tool = ROOT / "k8s-setup/scripts/rotate-jwt-secret.py"
        key_file = self.root / "replacement"
        key_file.write_text("replacement-signing-key-at-least-32-bytes")
        key_file.chmod(0o600)
        original = "original-signing-key-at-least-32-bytes"
        self.seed(original)
        for mode in ["deployment-missing", "patch-conflict"]:
            self.env["FAKE_MODE"] = mode
            result = subprocess.run(["python3", str(tool), "--key-file", str(key_file)], env=self.env,
                                    text=True, capture_output=True, timeout=10)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(self.stored(), original)
        self.env["FAKE_MODE"] = "rollout-failure"
        result = subprocess.run(["python3", str(tool), "--key-file", str(key_file)], env=self.env,
                                text=True, capture_output=True, timeout=10)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("rollout is incomplete", result.stderr)
        self.assertEqual(self.stored(), key_file.read_text())
        self.env["FAKE_MODE"] = ""
        result = subprocess.run(["python3", str(tool), "--key-file", str(key_file)], env=self.env,
                                text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_read_or_create_denial_is_not_treated_as_success(self):
        for mode in ["get-denied", "create-denied"]:
            with self.subTest(mode=mode):
                result = self.run_helper(mode=mode)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.store.exists())

    def test_concurrent_creation_reuses_valid_winning_secret(self):
        result = self.run_helper(mode="race")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("already configured", result.stdout)
        self.assertGreaterEqual(len(self.stored()), 32)

    def test_backend_requires_the_provisioned_secret(self):
        backend = named_container("main-service")
        env = {item["name"]: item for item in backend["env"]}
        self.assertEqual(env["JWT_SECRET"]["valueFrom"]["secretKeyRef"],
                         {"name": "gym-jinni-auth", "key": "JWT_SECRET", "optional": False})
        self.assertNotIn("APP_ENV", env)

    def test_container_selection_survives_reordering_and_reformatting(self):
        resources = list(yaml.safe_load_all(MANIFEST.read_text()))
        deployment = next(item for item in resources if item["kind"] == "Deployment" and item["metadata"]["name"] == "main-service")
        spec = deployment["spec"]["template"]["spec"]
        spec["containers"].insert(0, {"name": "sidecar", "image": "fixture"})
        spec["initContainers"].reverse()
        reformatted = self.root / "deployment.yml"
        reformatted.write_text(yaml.safe_dump_all(resources, sort_keys=True, indent=4))
        self.assertEqual(named_container("main-service", path=reformatted)["name"], "main-service")
        self.assertEqual(named_container("run-migrations", init=True, path=reformatted),
                         named_container("run-migrations", init=True))

    def test_actual_init_command_preserves_migrate_exit_status(self):
        migration = named_container("run-migrations", init=True)
        command = migration["command"]
        args = migration["args"]
        executable = self.root / "migrate"
        executable.write_text("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$MIGRATE_ARGS\"\nexit \"$MIGRATE_STATUS\"\n")
        executable.chmod(0o700)
        self.env.update(DBURL="postgresql://fixture/db?sslmode=disable&search_path=public",
                        MIGRATE_ARGS=str(self.root / "migrate-args"))
        for status in [0, 42]:
            with self.subTest(status=status):
                self.env["MIGRATE_STATUS"] = str(status)
                result = subprocess.run(command + args, env=self.env,
                                        text=True, capture_output=True, timeout=10)
                self.assertEqual(result.returncode, status)
                recorded_args = (self.root / "migrate-args").read_text().splitlines()
                self.assertEqual(recorded_args, ["-path", "/app/service/mig", "-database", self.env["DBURL"], "up"])


if __name__ == "__main__":
    unittest.main()
