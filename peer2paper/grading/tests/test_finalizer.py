from __future__ import annotations

import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import unittest


GRADING_ROOT = Path(__file__).resolve().parents[1]


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


finalizer = load_module("peer2paper_finalizer", GRADING_ROOT / "finalize.py")
grader = load_module("peer2paper_grader_for_finalizer", GRADING_ROOT / "service.py")


class FinalizerTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary.name)
        self.run_id = "run-01"
        self.case_id = "synthetic-orchard-01"
        self.run_root = self.root / "project" / "peer2paper" / "audits" / self.run_id
        self.run_root.mkdir(parents=True)
        self.contract_path = self.root / "validation-question-contract.json"
        self.contract = {
            "schema_version": "1.0.0",
            "case_id": self.case_id,
            "targets": [
                {
                    "id": "sensor-event-count",
                    "question": "How many events did the synthetic orchard sensor emit?",
                    "answer_type": "integer",
                    "producer_stage": "study_case",
                    "evidence_requirement": "Use the sealed dataset map.",
                },
                {
                    "id": "mean-moisture",
                    "question": "What is the synthetic mean soil-moisture reading?",
                    "answer_type": "number",
                    "producer_stage": "reproduction",
                    "evidence_requirement": "Use the sealed reproduction package.",
                },
                {
                    "id": "slope",
                    "question": "What is the synthetic calibration slope?",
                    "answer_type": "number",
                    "producer_stage": "reproduction",
                    "evidence_requirement": "Use the sealed secondary checkpoint.",
                },
                {
                    "id": "trend",
                    "question": "What direction does the synthetic trend take?",
                    "answer_type": "string",
                    "producer_stage": "reproduction",
                    "evidence_requirement": "Use the sealed secondary checkpoint.",
                },
            ],
        }
        self.write_json(self.contract_path, self.contract)

        self.dataset_path = self.run_root / "study-case" / "dataset-maps.json"
        self.reproduction_path = self.run_root / "reproduction" / "reproduction-package.json"
        self.write_json(self.dataset_path, {"sensor_summary": {"event_count": 987654321}})
        self.write_json(
            self.reproduction_path,
            {"synthetic_metrics": {"mean_moisture": 0.73142}},
        )
        self.write_sidecars()

        self.keys = self.root / "private" / "keys"
        self.receipts = self.root / "private" / "receipts"
        self.keys.mkdir(mode=0o700, parents=True)
        self.receipts.mkdir(mode=0o700, parents=True)
        key = {
            "schema_version": "1.0.0",
            "key_id": "synthetic-orchard-01-v1",
            "case_id": self.case_id,
            "visibility": "grader_only",
            "targets": [
                {"id": "sensor-event-count", "expected": 987654321, "comparison": "exact"},
                {"id": "mean-moisture", "expected": 0.731, "comparison": "three_decimal_places"},
                {"id": "slope", "expected": 1.625, "comparison": "three_decimal_places"},
                {"id": "trend", "expected": "Increase", "comparison": "case_insensitive_exact"},
            ],
        }
        key_path = self.keys / f"{self.case_id}.json"
        self.write_json(key_path, key)
        key_path.chmod(0o600)
        self.token = "t" * 32
        self.token_file = self.root / "private" / "service.token"
        self.token_file.write_text(self.token, encoding="utf-8")
        self.token_file.chmod(0o600)
        service = grader.GradeService(self.keys, self.receipts)
        self.server = grader.ThreadingHTTPServer(
            ("127.0.0.1", 0), grader.handler_factory(service, self.token)
        )
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.endpoint = f"http://127.0.0.1:{self.server.server_address[1]}/v1/grade"

    def tearDown(self) -> None:
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)
        self.temporary.cleanup()

    @staticmethod
    def write_json(path: Path, value: object) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(value, sort_keys=True), encoding="utf-8")

    @staticmethod
    def digest(path: Path) -> str:
        return hashlib.sha256(path.read_bytes()).hexdigest()

    def evidence(self, path: Path, pointer: str) -> dict[str, str]:
        relative = path.relative_to(self.run_root).as_posix()
        return {
            "artifact_uri": f"project://peer2paper/audits/{self.run_id}/{relative}",
            "artifact_sha256": self.digest(path),
            "json_pointer": pointer,
        }

    def write_sidecars(self) -> None:
        study_case = {
            "schema_version": "1.0.0",
            "case_id": self.case_id,
            "run_id": self.run_id,
            "producer_stage": "study_case",
            "observations": [
                {
                    "target_id": "sensor-event-count",
                    "value": 987654321,
                    "evidence": [self.evidence(self.dataset_path, "/sensor_summary/event_count")],
                }
            ],
        }
        reproduction = {
            "schema_version": "1.0.0",
            "case_id": self.case_id,
            "run_id": self.run_id,
            "producer_stage": "reproduction",
            "observations": [
                {
                    "target_id": "mean-moisture",
                    "value": 0.73142,
                    "evidence": [
                        self.evidence(
                            self.reproduction_path,
                            "/synthetic_metrics/mean_moisture",
                        )
                    ],
                }
            ],
        }
        self.write_json(
            self.run_root / "study-case" / "validation-observations.json", study_case
        )
        self.write_json(
            self.run_root / "reproduction" / "validation-observations.json", reproduction
        )

    def finalize(self, submit: bool = True) -> dict[str, object]:
        return finalizer.finalize_run(
            run_root=self.run_root,
            question_contract_path=self.contract_path,
            terminal_outcome="partial_reproduction",
            grader_url=self.endpoint if submit else None,
            token_file=self.token_file if submit else None,
        )

    def test_simulated_terminal_run_is_finalized_and_graded(self) -> None:
        result = self.finalize()
        self.assertEqual(result["status"], "graded")
        self.assertEqual(result["observation_count"], 2)
        self.assertEqual(result["missing_target_ids"], ["slope", "trend"])
        self.assertEqual(
            result["score"],
            {"points_awarded": 2, "points_possible": 4, "fraction": 0.5},
        )

        output_root = self.run_root / "validation"
        observations = json.loads((output_root / "observations.json").read_text())
        submission = json.loads((output_root / "grading-submission.json").read_text())
        receipt = json.loads((output_root / "grading-receipt.json").read_text())
        self.assertEqual(observations["terminal_outcome"], "partial_reproduction")
        self.assertEqual(len(submission["answers"]), 2)
        statuses = {item["target_id"]: item["status"] for item in receipt["targets"]}
        self.assertEqual(statuses["sensor-event-count"], "passed")
        self.assertEqual(statuses["mean-moisture"], "passed")
        self.assertEqual(statuses["slope"], "missing")
        self.assertEqual(statuses["trend"], "missing")
        self.assertFalse(receipt["expected_values_disclosed"])

    def test_identical_finalization_retry_is_idempotent(self) -> None:
        first = self.finalize()
        receipt_before = (self.run_root / "validation" / "grading-receipt.json").read_bytes()
        second = self.finalize()
        receipt_after = (self.run_root / "validation" / "grading-receipt.json").read_bytes()
        self.assertEqual(first, second)
        self.assertEqual(receipt_before, receipt_after)

    def test_finalize_only_mode_creates_no_receipt(self) -> None:
        result = self.finalize(submit=False)
        self.assertEqual(result["status"], "finalized")
        self.assertIsNone(result["receipt_path"])
        self.assertTrue((self.run_root / "validation" / "observations.json").is_file())
        self.assertTrue((self.run_root / "validation" / "grading-submission.json").is_file())
        self.assertFalse((self.run_root / "validation" / "grading-receipt.json").exists())

    def test_manual_command_finalizes_a_terminal_run(self) -> None:
        completed = subprocess.run(
            [
                sys.executable,
                str(GRADING_ROOT / "finalize.py"),
                "--run-root",
                str(self.run_root),
                "--question-contract",
                str(self.contract_path),
                "--terminal-outcome",
                "partial_reproduction",
            ],
            check=False,
            capture_output=True,
            text=True,
            timeout=10,
        )
        self.assertEqual(completed.returncode, 0, completed.stderr)
        result = json.loads(completed.stdout)
        self.assertEqual(result["status"], "finalized")
        self.assertEqual(result["observation_count"], 2)
        self.assertEqual(result["missing_target_ids"], ["slope", "trend"])

    def test_evidence_hash_mismatch_blocks_finalization(self) -> None:
        sidecar_path = self.run_root / "reproduction" / "validation-observations.json"
        sidecar = json.loads(sidecar_path.read_text())
        sidecar["observations"][0]["evidence"][0]["artifact_sha256"] = "a" * 64
        self.write_json(sidecar_path, sidecar)
        with self.assertRaisesRegex(finalizer.FinalizationError, "hash mismatch"):
            self.finalize(submit=False)
        self.assertFalse((self.run_root / "validation").exists())

    def test_json_pointer_value_mismatch_blocks_finalization(self) -> None:
        sidecar_path = self.run_root / "reproduction" / "validation-observations.json"
        sidecar = json.loads(sidecar_path.read_text())
        sidecar["observations"][0]["value"] = 0.5
        self.write_json(sidecar_path, sidecar)
        with self.assertRaisesRegex(finalizer.FinalizationError, "does not match observation"):
            self.finalize(submit=False)

    def test_changed_retry_cannot_overwrite_sealed_submission(self) -> None:
        self.finalize(submit=False)
        self.write_json(
            self.reproduction_path,
            {"synthetic_metrics": {"mean_moisture": 0.5}},
        )
        self.write_sidecars()
        sidecar_path = self.run_root / "reproduction" / "validation-observations.json"
        sidecar = json.loads(sidecar_path.read_text())
        sidecar["observations"][0]["value"] = 0.5
        self.write_json(sidecar_path, sidecar)
        with self.assertRaisesRegex(finalizer.FinalizationError, "refusing to overwrite"):
            self.finalize(submit=False)

    def test_unregistered_observation_is_rejected(self) -> None:
        sidecar_path = self.run_root / "reproduction" / "validation-observations.json"
        sidecar = json.loads(sidecar_path.read_text())
        sidecar["observations"][0]["target_id"] = "not-in-contract"
        self.write_json(sidecar_path, sidecar)
        with self.assertRaisesRegex(finalizer.FinalizationError, "not in the visible contract"):
            self.finalize(submit=False)

    def test_visible_contract_rejects_expected_values(self) -> None:
        contract = json.loads(self.contract_path.read_text())
        contract["targets"][0]["expected"] = 987654321
        self.write_json(self.contract_path, contract)
        with self.assertRaisesRegex(finalizer.FinalizationError, "unsupported fields: expected"):
            self.finalize(submit=False)

    def test_non_loopback_grader_url_is_rejected(self) -> None:
        with self.assertRaisesRegex(finalizer.FinalizationError, "loopback"):
            finalizer.validate_grader_url("https://grader.example/v1/grade")


if __name__ == "__main__":
    unittest.main()
