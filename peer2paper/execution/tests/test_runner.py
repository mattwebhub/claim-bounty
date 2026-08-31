from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock


RUNNER_PATH = Path(__file__).resolve().parents[1] / "runner.py"
SPEC = importlib.util.spec_from_file_location("peer2paper_runner", RUNNER_PATH)
assert SPEC and SPEC.loader
runner = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(runner)


class RunnerContractTests(unittest.TestCase):
    def manifest(self, attempt_directory: Path, **overrides: object) -> dict[str, object]:
        value: dict[str, object] = {
            "protocolVersion": "1.0.0",
            "runId": "test-run",
            "attemptId": "test-attempt",
            "attemptDirectory": str(attempt_directory),
            "workingDirectory": str(attempt_directory / "work"),
            "command": [sys.executable, "-c", "print('ok')"],
            "runtime": {"kind": "Python", "requiredPackages": []},
            "network": "host-policy",
        }
        value.update(overrides)
        return value

    def test_operational_limits_and_timeout_have_defaults(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            manifest = self.manifest(Path(temporary))
            self.assertEqual(runner.effective_timeout_seconds(manifest), 600)
            self.assertEqual(
                runner.effective_limits(manifest),
                {
                    "maximumMemoryBytes": 8 * 1024**3,
                    "maximumWorkingStorageBytes": 5 * 1024**3,
                },
            )

    def test_manifest_accepts_operational_defaults(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            self.assertEqual(runner.validate_manifest(self.manifest(attempt)), [])

    def test_manifest_rejects_nonpositive_timeout(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            failures = runner.validate_manifest(self.manifest(attempt, timeoutSeconds=0))
            self.assertEqual(failures[0]["class"], "invalid_manifest")
            self.assertIn("timeoutSeconds", failures[0]["message"])

    def test_manifest_requires_attempt_local_working_directory(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            outside = Path(temporary) / "source"
            attempt.mkdir()
            outside.mkdir()
            failures = runner.validate_manifest(
                self.manifest(attempt, workingDirectory=str(outside))
            )
            self.assertEqual(failures[0]["class"], "invalid_manifest")
            self.assertIn("inside attemptDirectory", failures[0]["message"])

    def test_typographic_quotes_are_rejected_before_r_execution(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "wrapper.R"
            source.write_text("setwd(\u201c/tmp/example\u201d)\n", encoding="utf-8")
            failure = runner.unsafe_r_source_failure(source)
            self.assertIsNotNone(failure)
            self.assertEqual(failure["class"], "invalid_wrapper")
            self.assertIn("typographic quote", failure["message"])

    def test_ascii_quoted_r_wrapper_passes_static_validation(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "wrapper.R"
            source.write_text('setwd("/tmp/example")\n', encoding="utf-8")
            self.assertIsNone(runner.unsafe_r_source_failure(source))

    def test_path_identity_detects_source_change(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "source"
            source.mkdir()
            script = source / "analysis.R"
            script.write_text("x <- 1\n", encoding="utf-8")
            before = runner.path_identity(source)
            script.write_text("x <- 2\n", encoding="utf-8")
            after = runner.path_identity(source)
            self.assertNotEqual(before["sha256"], after["sha256"])

    def test_execute_stops_process_after_timeout(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            work = attempt / "work"
            work.mkdir()
            manifest = self.manifest(
                attempt,
                command=[sys.executable, "-c", "import time; time.sleep(5)"],
                timeoutSeconds=1,
            )
            readiness = {
                "status": "ready",
                "checks": {
                    "runner": {"runnerSha256": "test"},
                    "manifestSha256": "manifest",
                    "environmentSha256": "environment",
                    "inputIdentities": [],
                },
                "failures": [],
            }
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner, "sandbox_command", side_effect=lambda value, directory: list(value["command"])
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["status"], "failed")
            self.assertEqual(record["terminationReason"], "timeout_exceeded")
            self.assertLess(record["durationSeconds"], 4)
            self.assertEqual(record["timeoutSeconds"], 1)
            self.assertRegex(record["commandSha256"], r"^[0-9a-f]{64}$")
            self.assertIn("outputIdentities", record)


if __name__ == "__main__":
    unittest.main()
