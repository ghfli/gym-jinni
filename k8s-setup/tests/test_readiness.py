"""Exercise the actual PostgreSQL wait command without a database dependency."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from manifest_utils import backend_pod_spec, named_container


class DatabaseReadinessTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = dict(os.environ, PATH=f"{self.root}:{os.environ['PATH']}")

    def test_database_wait_retries_until_ready_and_has_a_bound(self):
        names = [item["name"] for item in backend_pod_spec()["initContainers"]]
        self.assertLess(names.index("wait-for-postgres"), names.index("run-migrations"))
        wait = named_container("wait-for-postgres", init=True)
        executable = self.root / "pg_isready"
        executable.write_text("#!/bin/sh\nn=0\n[ ! -f \"$WAIT_COUNT\" ] || n=$(cat \"$WAIT_COUNT\")\nn=$((n + 1))\nprintf '%s' \"$n\" > \"$WAIT_COUNT\"\n[ \"$n\" -ge \"$READY_AFTER\" ]\n")
        executable.chmod(0o700)
        sleeper = self.root / "sleep"
        sleeper.write_text("#!/bin/sh\nexit 0\n")
        sleeper.chmod(0o700)
        self.env.update(DBURL="postgresql://fixture/db", WAIT_COUNT=str(self.root / "wait-count"))
        for ready_after, expected_code, expected_checks in [(3, 0, 3), (100, 1, 60)]:
            with self.subTest(ready_after=ready_after):
                (self.root / "wait-count").unlink(missing_ok=True)
                self.env["READY_AFTER"] = str(ready_after)
                result = subprocess.run(wait["command"] + wait["args"], env=self.env,
                                        text=True, capture_output=True, timeout=10)
                self.assertEqual(result.returncode, expected_code, result.stderr)
                self.assertEqual(int((self.root / "wait-count").read_text()), expected_checks)


if __name__ == "__main__":
    unittest.main()
