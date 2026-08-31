from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import tempfile
import threading
import unittest
from urllib import error, request


SERVICE_PATH = Path(__file__).resolve().parents[1] / "service.py"
SPEC = importlib.util.spec_from_file_location("peer2paper_grader", SERVICE_PATH)
assert SPEC and SPEC.loader
grader = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(grader)


class GradeServiceTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        root = Path(self.temporary.name)
        self.keys = root / "keys"
        self.receipts = root / "receipts"
        self.keys.mkdir(mode=0o700)
        self.receipts.mkdir(mode=0o700)
        self.case_id = "synthetic-orchard-01"
        self.key = {
            "schema_version": "1.0.0",
            "key_id": "key-synthetic-orchard-01-v1",
            "case_id": self.case_id,
            "visibility": "grader_only",
            "targets": [
                {
                    "id": "sensor-event-count",
                    "expected": 987654321,
                    "comparison": "exact",
                    "points": 1,
                },
                {
                    "id": "mean-moisture",
                    "expected": 0.731,
                    "comparison": "three_decimal_places",
                    "points": 2,
                },
                {
                    "id": "trend",
                    "expected": "Increase",
                    "comparison": "case_insensitive_exact",
                    "points": 1,
                },
            ],
        }
        key_path = self.keys / f"{self.case_id}.json"
        key_path.write_text(
            json.dumps(self.key), encoding="utf-8"
        )
        key_path.chmod(0o600)
        self.service = grader.GradeService(self.keys, self.receipts)

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def evidence(self) -> list[dict[str, str]]:
        return [
            {
                "artifact_uri": "project://peer2paper/audits/run-01/result.json",
                "artifact_sha256": "a" * 64,
                "json_pointer": "/result",
            }
        ]

    def submission(self, **overrides: object) -> dict[str, object]:
        value: dict[str, object] = {
            "schema_version": "1.0.0",
            "submission_id": "submission-01",
            "case_id": self.case_id,
            "run_id": "run-01",
            "phase": "final",
            "answers": [
                {
                    "target_id": "sensor-event-count",
                    "value": 987654321,
                    "evidence": self.evidence(),
                },
                {
                    "target_id": "mean-moisture",
                    "value": 0.73142,
                    "evidence": self.evidence(),
                },
                {
                    "target_id": "trend",
                    "value": "increase",
                    "evidence": self.evidence(),
                },
            ],
        }
        value.update(overrides)
        return value

    def test_grades_without_disclosing_expected_values(self) -> None:
        receipt = self.service.grade(self.submission())
        self.assertEqual(receipt["status"], "passed")
        self.assertEqual(
            receipt["score"],
            {"points_awarded": 4, "points_possible": 4, "fraction": 1.0},
        )
        self.assertTrue(all(item["status"] == "passed" for item in receipt["targets"]))
        encoded = json.dumps(receipt, sort_keys=True)
        self.assertNotIn("987654321", encoded)
        self.assertNotIn("0.731", encoded)
        self.assertNotIn('"Increase"', encoded)
        self.assertFalse(receipt["expected_values_disclosed"])

    def test_missing_answer_is_scored_without_revealing_expected(self) -> None:
        submission = self.submission()
        submission["answers"] = submission["answers"][:1]
        receipt = self.service.grade(submission)
        statuses = {item["target_id"]: item["status"] for item in receipt["targets"]}
        self.assertEqual(statuses["sensor-event-count"], "passed")
        self.assertEqual(statuses["mean-moisture"], "missing")
        self.assertEqual(statuses["trend"], "missing")
        self.assertEqual(receipt["status"], "failed")

    def test_identical_retry_is_idempotent(self) -> None:
        first = self.service.grade(self.submission())
        second = self.service.grade(self.submission())
        self.assertEqual(first, second)

    def test_changed_retry_for_same_run_is_rejected(self) -> None:
        self.service.grade(self.submission())
        changed = self.submission()
        changed["answers"][0]["value"] = 987654320
        with self.assertRaises(grader.AlreadyGraded):
            self.service.grade(changed)

    def test_unregistered_target_is_rejected(self) -> None:
        submission = self.submission()
        submission["answers"].append(
            {
                "target_id": "not-in-key",
                "value": 1,
                "evidence": self.evidence(),
            }
        )
        with self.assertRaises(grader.InvalidSubmission):
            self.service.grade(submission)

    def test_case_identifier_cannot_traverse_key_root(self) -> None:
        with self.assertRaises(grader.InvalidSubmission):
            self.service.grade(self.submission(case_id="../synthetic-orchard-01"))

    def test_evidence_hash_is_required(self) -> None:
        submission = self.submission()
        del submission["answers"][0]["evidence"][0]["artifact_sha256"]
        with self.assertRaises(grader.InvalidSubmission):
            self.service.grade(submission)

    def test_project_local_key_root_is_rejected_by_secure_startup(self) -> None:
        token = grader.PROJECT_ROOT / ".tmp" / "grader-test-token"
        key_root = grader.PROJECT_ROOT / ".tmp" / "grader-test-keys"
        receipt_root = grader.PROJECT_ROOT / ".tmp" / "grader-test-receipts"
        for directory in (key_root, receipt_root):
            directory.mkdir(mode=0o700, parents=True, exist_ok=True)
        token.write_text("x" * 32, encoding="utf-8")
        token.chmod(0o600)
        try:
            with self.assertRaises(grader.GraderConfigurationError):
                grader.validate_service_paths(
                    key_root,
                    receipt_root,
                    token,
                    allow_insecure_development=False,
                )
        finally:
            token.unlink(missing_ok=True)
            key_root.rmdir()
            receipt_root.rmdir()

    def test_http_endpoint_requires_token_and_returns_receipt(self) -> None:
        token = "t" * 32
        server = grader.ThreadingHTTPServer(
            ("127.0.0.1", 0), grader.handler_factory(self.service, token)
        )
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        endpoint = f"http://127.0.0.1:{server.server_address[1]}/v1/grade"
        body = json.dumps(self.submission()).encode("utf-8")
        try:
            unauthenticated = request.Request(
                endpoint,
                data=body,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with self.assertRaises(error.HTTPError) as rejected:
                request.urlopen(unauthenticated, timeout=2)
            self.assertEqual(rejected.exception.code, 401)

            authenticated = request.Request(
                endpoint,
                data=body,
                headers={
                    "Authorization": f"Bearer {token}",
                    "Content-Type": "application/json",
                },
                method="POST",
            )
            with request.urlopen(authenticated, timeout=2) as response:
                receipt = json.loads(response.read())
            self.assertEqual(receipt["status"], "passed")
            self.assertFalse(receipt["expected_values_disclosed"])
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)


if __name__ == "__main__":
    unittest.main()
