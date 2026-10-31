#!/usr/bin/env python3
"""Explicit operator rotation: preserve the Secret, then roll/recheck backend replicas.

Use a protected key file. This invalidates existing tokens after the rollout.
The normal ensure helper never rotates a key implicitly.
"""
import argparse
import base64
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile

NAMESPACE = "gym-jinni"
SECRET = "gym-jinni-auth"
DEPLOYMENT = "main-service"


def kubectl(*args, timeout=30):
    result = subprocess.run(["kubectl", *args, "-n", NAMESPACE],
                            text=True, capture_output=True, timeout=timeout)
    if result.returncode:
        raise RuntimeError(f"kubectl {args[0]} failed; check cluster access, resource version and rollout events")
    return result.stdout


def rotate(key_file):
    if (os.environ.get("KUBE_NAMESPACE") or NAMESPACE) != NAMESPACE:
        raise ValueError("KUBE_NAMESPACE must be gym-jinni to match the manifests")
    path = Path(key_file)
    if stat.S_IMODE(path.stat().st_mode) & 0o077:
        raise ValueError("Key file must be private (no group/other permissions)")
    key = path.read_bytes()
    if len(key) < 32 or b"\0" in key or key == b"gym-jinni-dev-secret-change-in-prod":
        raise ValueError("Key must be at least 32 bytes, without NUL bytes or the development default")
    secret = json.loads(kubectl("get", "secret", SECRET, "-o", "json"))
    # Confirm a rollout target exists before making any change.
    kubectl("get", "deployment", DEPLOYMENT, "-o", "json")
    current = base64.b64decode(secret.get("data", {}).get("JWT_SECRET", ""), validate=True)
    changed = current != key
    if changed:
        patch = [{"op": "test", "path": "/metadata/resourceVersion", "value": secret["metadata"]["resourceVersion"]}]
        if "data" not in secret:
            patch.append({"op": "add", "path": "/data", "value": {}})
        patch.append({"op": "add", "path": "/data/JWT_SECRET", "value": base64.b64encode(key).decode("ascii")})
        with tempfile.TemporaryDirectory(prefix="jwt-rotation-") as directory:
            patch_file = Path(directory) / "patch.json"
            with open(patch_file, "x", opener=lambda p, flags: os.open(p, flags, 0o600)) as stream:
                json.dump(patch, stream)
            kubectl("patch", "secret", SECRET, "--type=json", "--patch-file", str(patch_file))
        print("JWT signing key updated without deleting the Secret.", flush=True)
    else:
        print("Key already matches; completing the backend rollout.", flush=True)
    try:
        kubectl("rollout", "restart", "deployment/" + DEPLOYMENT)
        kubectl("rollout", "status", "deployment/" + DEPLOYMENT, "--timeout=300s", timeout=310)
    except (RuntimeError, subprocess.TimeoutExpired) as error:
        raise RuntimeError("Secret has the requested key but rollout is incomplete. Fix the rollout and retry the same rotation command; do not delete the Secret.") from error
    print("Backend rotation rollout complete; align CI JWT_SECRET with this key. Existing tokens require login.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--key-file", required=True)
    args = parser.parse_args()
    try:
        rotate(args.key_file)
    except (OSError, ValueError, KeyError, RuntimeError, subprocess.TimeoutExpired) as error:
        # Do not dump API responses, key bytes or patch contents.
        print(str(error) if isinstance(error, (ValueError, RuntimeError)) else "Rotation failed; verify the protected key file, cluster resources and access.", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
