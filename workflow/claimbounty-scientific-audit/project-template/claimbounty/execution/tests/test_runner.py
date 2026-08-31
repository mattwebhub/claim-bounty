from __future__ import annotations

import importlib.util
import hashlib
import json
import platform
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest import mock


RUNNER_PATH = Path(__file__).resolve().parents[1] / "runner.py"
SPEC = importlib.util.spec_from_file_location("claimbounty_runner", RUNNER_PATH)
assert SPEC and SPEC.loader
runner = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(runner)


class RunnerContractTests(unittest.TestCase):
    _runtime_read_roots: list[dict[str, str]] | None = None

    @classmethod
    def runtime_read_roots(cls) -> list[dict[str, str]]:
        if cls._runtime_read_roots is None:
            version = f"python{sys.version_info.major}.{sys.version_info.minor}"
            prefix = Path(sys.base_prefix).resolve()
            candidates = [prefix / "lib" / version, prefix / "lib" / f"lib{version}.dylib"]
            cls._runtime_read_roots = []
            for candidate in candidates:
                if not candidate.exists():
                    continue
                candidate = candidate.resolve()
                identity = runner.path_identity(candidate)
                cls._runtime_read_roots.append(
                    {
                        "path": str(candidate),
                        "kind": identity["kind"],
                        "sha256": identity["sha256"],
                    }
                )
        return [dict(root) for root in cls._runtime_read_roots]

    def execution_policy(
        self,
        directory: Path,
        *,
        sandbox_overrides: dict[str, object] | None = None,
        runtime_read_roots: list[dict[str, str]] | None = None,
        **policy_overrides: object,
    ) -> dict[str, str]:
        executable = Path(sys.executable).resolve()
        policy = {
            "schemaVersion": "1.2.0",
            "policyVersion": "test-policy-v1",
            "runClass": "manual_local_operator",
            "releaseScope": "internal",
            "outputRoot": str(directory.parent.resolve()),
            "runtime": {
                "kind": "Python",
                "command": "python3",
                "executable": str(executable),
                "executableSha256": runner.sha256_file(executable),
                "version": platform.python_version(),
                "requiredPackages": [],
                "readRoots": self.runtime_read_roots() + (runtime_read_roots or []),
            },
            "sandbox": {
                "isolationRequired": True,
                "networkDuringAnalysis": "disabled",
                "dependencyAcquisition": "disabled",
                "expandArchivesAutomatically": False,
                **(sandbox_overrides or {}),
            },
            "limits": {
                "timeoutSeconds": 600,
                "maximumMemoryBytes": 8 * 1024**3,
                "maximumWorkingStorageBytes": 5 * 1024**3,
            },
            "mounts": [],
            **policy_overrides,
        }
        path = directory / "execution-policy.json"
        contents = json.dumps(policy, sort_keys=True).encode("utf-8")
        path.write_bytes(contents)
        return {"path": str(path), "sha256": hashlib.sha256(contents).hexdigest()}

    def manifest(self, attempt_directory: Path, **overrides: object) -> dict[str, object]:
        value: dict[str, object] = {
            "protocolVersion": "1.2.0",
            "runId": "test-run",
            "attemptId": "test-attempt",
            "attemptDirectory": str(attempt_directory),
            "workingDirectory": str(attempt_directory / "work"),
            "command": ["python3", "-c", "print('ok')"],
            "executionPolicy": self.execution_policy(attempt_directory),
        }
        value.update(overrides)
        return value

    def ready_for_execute(self, manifest: dict[str, object]) -> dict[str, object]:
        _, policy_identity = runner.frozen_execution_policy(manifest)
        return {
            "status": "ready",
            "checks": {
                "runner": {"runnerSha256": "test"},
                "executionPolicy": policy_identity,
                "networkIsolationVerified": True,
                "hostFileReadIsolationVerified": True,
                "manifestSha256": "manifest",
                "environmentSha256": "environment",
                "inputIdentities": [],
            },
            "failures": [],
        }

    def test_operational_limits_and_timeout_are_frozen_in_profile(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            manifest = self.manifest(Path(temporary))
            policy, _ = runner.frozen_execution_policy(manifest)
            self.assertEqual(runner.effective_timeout_seconds(policy), 600)
            self.assertEqual(
                runner.effective_limits(policy),
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

    def test_manifest_rejects_limit_and_mount_weakening(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            failures = runner.validate_manifest(
                self.manifest(
                    attempt,
                    timeoutSeconds=9999,
                    limits={"maximumMemoryBytes": 2**63},
                    mounts=[{"path": "/", "mode": "read-only"}],
                    environment={"AWS_SECRET_ACCESS_KEY": "injected"},
                )
            )
            self.assertEqual(failures[0]["class"], "invalid_manifest")
            self.assertIn("environment", failures[0]["message"])
            self.assertIn("limits", failures[0]["message"])
            self.assertIn("mounts", failures[0]["message"])
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

    def test_manifest_rejects_independent_network_weakening(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            failures = runner.validate_manifest(self.manifest(attempt, network="host-policy"))
            self.assertEqual(failures[0]["class"], "invalid_manifest")
            self.assertIn("derived from executionPolicy", failures[0]["message"])

    def test_manifest_selected_hostile_version_command_is_never_executed(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            marker = attempt / "hostile-version-ran"
            hostile = attempt / "python3-hostile"
            hostile.write_text(
                f"#!/bin/sh\nprintf ran > {marker}\n",
                encoding="utf-8",
            )
            hostile.chmod(0o755)
            readiness = runner.preflight(
                self.manifest(attempt, command=[str(hostile), "--version"])
            )
            self.assertEqual(readiness["status"], "blocked")
            self.assertFalse(marker.exists())
            self.assertIn("profile runtime alias", readiness["failures"][-1]["message"])

    def test_minimal_environment_does_not_inherit_secrets(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            manifest = self.manifest(attempt)
            policy, _ = runner.frozen_execution_policy(manifest)
            with mock.patch.dict(
                runner.os.environ,
                {"CLAIMBOUNTY_HOST_SECRET": "must-not-cross", "AWS_SECRET_ACCESS_KEY": "no"},
            ):
                environment = runner.minimal_environment(attempt, policy)
            self.assertNotIn("CLAIMBOUNTY_HOST_SECRET", environment)
            self.assertNotIn("AWS_SECRET_ACCESS_KEY", environment)
            self.assertEqual(set(environment), {"HOME", "LANG", "LC_ALL", "PATH", "TMPDIR"})

    def test_execute_does_not_pass_inherited_secret_to_scientific_process(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            manifest = self.manifest(
                attempt,
                command=[
                    "python3",
                    "-c",
                    (
                        "import os,sys; "
                        "sys.exit(9 if os.getenv('CLAIMBOUNTY_HOST_SECRET') else 0)"
                    ),
                ],
            )
            _, policy_identity = runner.frozen_execution_policy(manifest)
            readiness = {
                "status": "ready",
                "checks": {
                    "runner": {"runnerSha256": "test"},
                    "executionPolicy": policy_identity,
                    "networkIsolationVerified": True,
                    "hostFileReadIsolationVerified": True,
                    "manifestSha256": "manifest",
                    "environmentSha256": "environment",
                    "inputIdentities": [],
                },
                "failures": [],
            }
            with mock.patch.dict(
                runner.os.environ,
                {"CLAIMBOUNTY_HOST_SECRET": "must-not-cross"},
            ), mock.patch.object(
                runner, "preflight", return_value=readiness
            ), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["status"], "completed", record)
            self.assertEqual(record["exitCode"], 0)

    def test_frozen_policy_digest_must_match(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            manifest = self.manifest(attempt)
            manifest["executionPolicy"]["sha256"] = "0" * 64
            readiness = runner.preflight(manifest)
            self.assertEqual(readiness["status"], "blocked")
            self.assertIn("does not match", readiness["failures"][0]["message"])

    def test_frozen_policy_cannot_enable_host_network(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            manifest = self.manifest(attempt)
            manifest["executionPolicy"] = self.execution_policy(
                attempt,
                sandbox_overrides={"networkDuringAnalysis": "host-policy"},
            )
            readiness = runner.preflight(manifest)
            self.assertEqual(readiness["status"], "blocked")
            self.assertIn("must disable network", readiness["failures"][0]["message"])

    def test_frozen_policy_rejects_limit_above_runner_cap(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            manifest = self.manifest(attempt)
            manifest["executionPolicy"] = self.execution_policy(
                attempt,
                limits={
                    "timeoutSeconds": 601,
                    "maximumMemoryBytes": 8 * 1024**3,
                    "maximumWorkingStorageBytes": 5 * 1024**3,
                },
            )
            readiness = runner.preflight(manifest)
            self.assertEqual(readiness["status"], "blocked")
            self.assertIn("safety cap", readiness["failures"][0]["message"])

    def test_frozen_policy_rejects_broad_host_mount(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary)
            (attempt / "work").mkdir()
            manifest = self.manifest(attempt)
            manifest["executionPolicy"] = self.execution_policy(
                attempt,
                mounts=[
                    {
                        "path": "/",
                        "mode": "read-only",
                        "verifyUnchanged": True,
                        "sha256": "0" * 64,
                    }
                ],
            )
            readiness = runner.preflight(manifest)
            self.assertEqual(readiness["status"], "blocked")
            self.assertIn("broader than", readiness["failures"][0]["message"])

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

    def test_r_parse_rejects_source_outside_allowed_roots_before_read_or_hash(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            work = attempt / "work"
            work.mkdir(parents=True)
            outside = root / "outside.R"
            outside.write_text("stop('must not be read')\n", encoding="utf-8")
            manifest = self.manifest(
                attempt,
                command=["Rscript", str(outside)],
            )
            policy, _ = runner.frozen_execution_policy(manifest)
            with mock.patch.object(
                runner,
                "unsafe_r_source_contents_failure",
                side_effect=AssertionError("outside source was read"),
            ):
                check, failure = runner.parse_r_command_source(
                    manifest,
                    {"executable": str(Path(sys.executable).resolve())},
                    work,
                    attempt,
                    policy,
                )
            self.assertEqual(check, {"sourceAccepted": False, "succeeded": False})
            self.assertEqual(failure["class"], "invalid_wrapper")
            self.assertNotIn(str(outside), failure["message"])

    def test_r_parse_rejects_attempt_local_symlink_before_read_or_hash(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            work = attempt / "work"
            work.mkdir(parents=True)
            outside = root / "outside.R"
            outside.write_text("stop('must not be read')\n", encoding="utf-8")
            linked = work / "analysis.R"
            linked.symlink_to(outside)
            manifest = self.manifest(
                attempt,
                command=["Rscript", str(linked)],
            )
            policy, _ = runner.frozen_execution_policy(manifest)
            with mock.patch.object(
                runner,
                "unsafe_r_source_contents_failure",
                side_effect=AssertionError("symlink target was read"),
            ):
                check, failure = runner.parse_r_command_source(
                    manifest,
                    {"executable": str(Path(sys.executable).resolve())},
                    work,
                    attempt,
                    policy,
                )
            self.assertEqual(check, {"sourceAccepted": False, "succeeded": False})
            self.assertEqual(failure["class"], "invalid_wrapper")
            self.assertNotIn("sourceSha256", check)

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
                command=["python3", "-c", "import time; time.sleep(5)"],
            )
            manifest["executionPolicy"] = self.execution_policy(
                attempt,
                limits={
                    "timeoutSeconds": 1,
                    "maximumMemoryBytes": 8 * 1024**3,
                    "maximumWorkingStorageBytes": 5 * 1024**3,
                },
            )
            _, policy_identity = runner.frozen_execution_policy(manifest)
            readiness = {
                "status": "ready",
                "checks": {
                    "runner": {"runnerSha256": "test"},
                    "executionPolicy": policy_identity,
                    "networkIsolationVerified": True,
                    "hostFileReadIsolationVerified": True,
                    "manifestSha256": "manifest",
                    "environmentSha256": "environment",
                    "inputIdentities": [],
                },
                "failures": [],
            }
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["status"], "failed")
            self.assertEqual(record["terminationReason"], "timeout_exceeded")
            self.assertLess(record["durationSeconds"], 4)
            self.assertEqual(record["timeoutSeconds"], 1)
            self.assertRegex(record["commandSha256"], r"^[0-9a-f]{64}$")
            self.assertIn("outputIdentities", record)

    def test_preexisting_stdout_symlink_blocks_without_touching_target(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            (attempt / "work").mkdir(parents=True)
            outside = root / "outside.txt"
            outside.write_text("unchanged\n", encoding="utf-8")
            (attempt / "stdout.log").symlink_to(outside)
            manifest = self.manifest(attempt)
            readiness = self.ready_for_execute(manifest)
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["status"], "blocked")
            self.assertIn("stdout.log", record["message"])
            self.assertEqual(outside.read_text(encoding="utf-8"), "unchanged\n")

    def test_attempt_record_symlink_escape_is_replaced_without_following(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            (attempt / "work").mkdir(parents=True)
            outside = root / "outside.json"
            outside.write_text("unchanged\n", encoding="utf-8")
            command = (
                "import os,sys; "
                "os.symlink(sys.argv[1], os.path.join(sys.argv[2], 'attempt-record.json'))"
            )
            manifest = self.manifest(
                attempt,
                command=["python3", "-c", command, str(outside), str(attempt)],
            )
            readiness = self.ready_for_execute(manifest)
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["status"], "failed", record)
            self.assertEqual(record["terminationReason"], "reserved_path_replaced")
            self.assertEqual(record["reservedPathViolations"], ["attempt-record.json"])
            self.assertEqual(outside.read_text(encoding="utf-8"), "unchanged\n")
            self.assertFalse((attempt / "attempt-record.json").is_symlink())
            persisted = json.loads((attempt / "attempt-record.json").read_text(encoding="utf-8"))
            self.assertEqual(persisted["terminationReason"], "reserved_path_replaced")

    def test_attempt_record_directory_replacement_still_emits_terminal_record(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            (attempt / "work").mkdir(parents=True)
            command = (
                "import os,sys; "
                "os.mkdir(os.path.join(sys.argv[1], 'attempt-record.json'))"
            )
            manifest = self.manifest(
                attempt,
                command=["python3", "-c", command, str(attempt)],
            )
            readiness = self.ready_for_execute(manifest)
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["status"], "failed", record)
            self.assertEqual(record["terminationReason"], "reserved_path_replaced")
            self.assertEqual(record["reservedPathViolations"], ["attempt-record.json"])
            self.assertEqual(len(record["reservedPathQuarantines"]), 1)
            quarantine = attempt / record["reservedPathQuarantines"][0]
            self.assertTrue(quarantine.is_dir())
            self.assertTrue((attempt / "attempt-record.json").is_file())
            persisted = json.loads((attempt / "attempt-record.json").read_text(encoding="utf-8"))
            self.assertEqual(persisted["reservedPathQuarantines"], record["reservedPathQuarantines"])

    def test_scientific_code_cannot_replace_log_write_boundaries_with_symlinks(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            (attempt / "work").mkdir(parents=True)
            outside_stdout = root / "outside-stdout.txt"
            outside_stderr = root / "outside-stderr.txt"
            outside_stdout.write_text("stdout unchanged\n", encoding="utf-8")
            outside_stderr.write_text("stderr unchanged\n", encoding="utf-8")
            command = (
                "import os,sys; a=sys.argv[1]; "
                "os.unlink(os.path.join(a, 'stdout.log')); "
                "os.unlink(os.path.join(a, 'stderr.log')); "
                "os.symlink(sys.argv[2], os.path.join(a, 'stdout.log')); "
                "os.symlink(sys.argv[3], os.path.join(a, 'stderr.log'))"
            )
            manifest = self.manifest(
                attempt,
                command=[
                    "python3",
                    "-c",
                    command,
                    str(attempt),
                    str(outside_stdout),
                    str(outside_stderr),
                ],
            )
            readiness = self.ready_for_execute(manifest)
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["status"], "failed", record)
            self.assertEqual(record["terminationReason"], "reserved_path_replaced")
            self.assertEqual(
                record["reservedPathViolations"], ["stderr.log", "stdout.log"]
            )
            self.assertEqual(outside_stdout.read_text(encoding="utf-8"), "stdout unchanged\n")
            self.assertEqual(outside_stderr.read_text(encoding="utf-8"), "stderr unchanged\n")

    def test_output_scan_rejects_symlink_without_hashing_target(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            attempt.mkdir()
            outside = root / "outside.txt"
            outside.write_text("must not be scanned\n", encoding="utf-8")
            (attempt / "result.txt").symlink_to(outside)
            identities, unsafe = runner.output_scan(attempt)
            self.assertEqual(identities, [])
            self.assertEqual(unsafe, [{"path": "result.txt", "kind": "symlink"}])

    def test_network_profile_symlink_is_rejected_without_following(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            attempt.mkdir()
            outside = root / "outside.sb"
            outside.write_text("unchanged\n", encoding="utf-8")
            (attempt / "network-deny.sb").symlink_to(outside)
            manifest = self.manifest(attempt)
            policy, _ = runner.frozen_execution_policy(manifest)
            with mock.patch.object(runner.platform, "system", return_value="Darwin"), mock.patch.object(
                runner.shutil, "which", return_value="/usr/bin/sandbox-exec"
            ):
                with self.assertRaisesRegex(ValueError, "unsafe type"):
                    runner.sandbox_command([sys.executable, "-c", "pass"], attempt, policy)
            self.assertEqual(outside.read_text(encoding="utf-8"), "unchanged\n")

    def test_probe_exception_persists_machine_readable_terminal_record(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            (attempt / "work").mkdir(parents=True)
            manifest = self.manifest(attempt)
            with mock.patch.object(runner, "preflight", side_effect=OSError("probe fault")):
                record = runner.execute(manifest)
            self.assertEqual(record["terminationReason"], "operational_failure")
            self.assertTrue(record["recordPersisted"])
            persisted = json.loads((attempt / "attempt-record.json").read_text(encoding="utf-8"))
            self.assertEqual(persisted["operationalFailure"]["type"], "OSError")

    def test_process_start_exception_persists_machine_readable_terminal_record(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            (attempt / "work").mkdir(parents=True)
            manifest = self.manifest(attempt)
            readiness = self.ready_for_execute(manifest)
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ), mock.patch.object(runner.subprocess, "Popen", side_effect=OSError("start fault")):
                record = runner.execute(manifest)
            self.assertEqual(record["terminationReason"], "operational_failure")
            self.assertTrue(record["recordPersisted"])
            self.assertTrue((attempt / "attempt-record.json").is_file())

    def test_output_scan_exception_persists_machine_readable_terminal_record(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            (attempt / "work").mkdir(parents=True)
            manifest = self.manifest(attempt)
            readiness = self.ready_for_execute(manifest)
            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ), mock.patch.object(runner, "directory_size", return_value=0), mock.patch.object(
                runner, "output_scan", side_effect=OSError("scan fault")
            ):
                record = runner.execute(manifest)
            self.assertEqual(record["terminationReason"], "operational_failure")
            self.assertTrue(record["recordPersisted"])
            self.assertTrue((attempt / "attempt-record.json").is_file())

    def test_primary_record_writer_failure_uses_independent_terminal_writer(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            (attempt / "work").mkdir(parents=True)
            manifest = self.manifest(attempt)
            readiness = self.ready_for_execute(manifest)
            original_writer = runner.atomic_write_reserved

            def fail_record_write(*args, **kwargs):
                if args[1] == "attempt-record.json":
                    raise OSError("record writer fault")
                return original_writer(*args, **kwargs)

            with mock.patch.object(runner, "preflight", return_value=readiness), mock.patch.object(
                runner,
                "sandbox_command",
                side_effect=lambda command, directory, policy: list(command),
            ), mock.patch.object(runner, "atomic_write_reserved", side_effect=fail_record_write):
                record = runner.execute(manifest)
            self.assertEqual(record["terminationReason"], "operational_failure")
            self.assertTrue(record["recordPersisted"])
            persisted = json.loads((attempt / "attempt-record.json").read_text(encoding="utf-8"))
            self.assertEqual(persisted["operationalFailure"]["type"], "OSError")

    def test_attempt_directory_cannot_escape_frozen_output_root(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            allowed = root / "allowed"
            (allowed / "work").mkdir(parents=True)
            manifest = self.manifest(allowed)
            outside = root.parent / f"claimbounty-outside-{root.name}"
            manifest["attemptDirectory"] = str(outside)
            manifest["workingDirectory"] = str(outside / "work")
            record = runner.execute(manifest)
            self.assertEqual(record["terminationReason"], "attempt_boundary_unavailable")
            self.assertFalse(outside.exists())

    def test_optional_output_cannot_create_arbitrary_host_parent(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            outside_parent = root.parent / f"claimbounty-output-{root.name}"
            with self.assertRaisesRegex(ValueError, "direct child"):
                runner.emit(
                    {"status": "blocked"},
                    outside_parent / "result.json",
                    root,
                )
            self.assertFalse(outside_parent.exists())

    def test_main_output_failure_replaces_attempt_record_with_terminal_failure(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            (attempt / "work").mkdir(parents=True)
            manifest = self.manifest(attempt)
            manifest_path = attempt / "manifest.json"
            output_path = attempt.parent / "runner-output.json"
            with mock.patch.object(
                runner.sys,
                "argv",
                [
                    "runner.py",
                    "run",
                    "--manifest",
                    str(manifest_path),
                    "--output",
                    str(output_path),
                ],
            ), mock.patch.object(runner, "load_manifest", return_value=manifest), mock.patch.object(
                runner,
                "execute",
                return_value={"status": "completed"},
            ), mock.patch.object(runner, "emit", side_effect=OSError("output fault")), mock.patch.object(
                runner.sys, "stdout"
            ):
                exit_code = runner.main()
            self.assertEqual(exit_code, 2)
            persisted = json.loads((attempt / "attempt-record.json").read_text(encoding="utf-8"))
            self.assertEqual(persisted["terminationReason"], "output_persistence_failure")

    @unittest.skipUnless(
        platform.system() == "Darwin" and shutil.which("sandbox-exec"),
        "sandbox-exec boundary is available only on the supported macOS host",
    )
    def test_live_sandbox_denies_network_and_host_file_escape(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            work = attempt / "work"
            work.mkdir(parents=True)
            host_file = root / "host-secret.txt"
            host_file.write_text("must not be readable\n", encoding="utf-8")
            command = (
                "import errno,socket,sys; denied=(errno.EPERM,errno.EACCES); "
                "s=socket.socket(); network=s.connect_ex(('127.0.0.1',9)); s.close(); "
                "\ntry:\n open(sys.argv[1], 'rb').read(); host=3"
                "\nexcept OSError as exc:\n host=0 if exc.errno in denied else 4"
                "\nsys.exit(0 if network in denied and host == 0 else 5)"
            )
            manifest = self.manifest(
                attempt,
                command=["python3", "-c", command, str(host_file)],
            )
            record = runner.execute(manifest)
            self.assertEqual(record["status"], "completed", record)
            self.assertEqual(
                record["isolationChecks"],
                {
                    "networkDenied": True,
                    "hostFileReadDenied": True,
                    "childProcessCreationDenied": True,
                },
            )

    @unittest.skipUnless(
        platform.system() == "Darwin" and shutil.which("sandbox-exec"),
        "sandbox-exec boundary is available only on the supported macOS host",
    )
    def test_live_sandbox_denies_fork_and_detached_child(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            attempt = Path(temporary) / "attempt"
            (attempt / "work").mkdir(parents=True)
            command = (
                "import errno,os,subprocess,sys; denied=(errno.EPERM,errno.EACCES); "
                "fork_denied=False; detach_denied=False"
                "\ntry:\n pid=os.fork()"
                "\nexcept OSError as exc:\n fork_denied=exc.errno in denied"
                "\nelse:\n"
                "  if pid == 0: os._exit(0)\n"
                "  os.waitpid(pid, 0)"
                "\ntry:\n child=subprocess.Popen([sys.executable,'-c','pass'], start_new_session=True)"
                "\nexcept OSError as exc:\n detach_denied=exc.errno in denied"
                "\nelse:\n child.terminate(); child.wait(); detach_denied=False"
                "\nsys.exit(0 if fork_denied and detach_denied else 3)"
            )
            manifest = self.manifest(attempt, command=["python3", "-c", command])
            record = runner.execute(manifest)
            self.assertEqual(record["status"], "completed", record)
            self.assertTrue(record["isolationChecks"]["childProcessCreationDenied"])

    @unittest.skipUnless(
        platform.system() == "Darwin" and shutil.which("sandbox-exec"),
        "sandbox-exec boundary is available only on the supported macOS host",
    )
    def test_live_sandbox_runtime_closure_does_not_expose_unrelated_sibling(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            attempt = root / "attempt"
            (attempt / "work").mkdir(parents=True)
            runtime_install = root / "runtime-install"
            closure = runtime_install / "closure"
            closure.mkdir(parents=True)
            allowed = closure / "runtime-data.txt"
            unrelated = runtime_install / "unrelated-study-input.txt"
            allowed.write_text("runtime closure\n", encoding="utf-8")
            unrelated.write_text("must be denied\n", encoding="utf-8")
            closure_identity = runner.path_identity(closure)
            command = (
                "import errno,sys; denied=(errno.EPERM,errno.EACCES); "
                "open(sys.argv[1], 'rb').read(); "
                "\ntry:\n open(sys.argv[2], 'rb').read(); result=3"
                "\nexcept OSError as exc:\n result=0 if exc.errno in denied else 4"
                "\nsys.exit(result)"
            )
            manifest = self.manifest(
                attempt,
                command=["python3", "-c", command, str(allowed), str(unrelated)],
            )
            manifest["executionPolicy"] = self.execution_policy(
                attempt,
                runtime_read_roots=[
                    {
                        "path": str(closure),
                        "kind": "directory",
                        "sha256": closure_identity["sha256"],
                    }
                ],
            )
            record = runner.execute(manifest)
            self.assertEqual(record["status"], "completed", record)


if __name__ == "__main__":
    unittest.main()
