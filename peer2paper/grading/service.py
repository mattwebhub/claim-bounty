#!/usr/bin/env python3
"""Local, score-only grading service for Peer2Paper validation runs.

The service owns access to grader-only answer keys. Audit agents submit candidate
answers and evidence references. Responses contain target-level pass/fail states
and scores, but never expected values or answer-key paths.
"""

from __future__ import annotations

import argparse
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation, ROUND_HALF_UP
import hashlib
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import hmac
import json
import math
import os
from pathlib import Path
import re
import stat
import tempfile
import threading
from typing import Any


SERVICE_NAME = "peer2paper-local-grader"
SERVICE_VERSION = "1.0.0"
SCHEMA_VERSION = "1.0.0"
MAX_REQUEST_BYTES = 1024 * 1024
IDENTIFIER_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
PROJECT_ROOT = Path(__file__).resolve().parents[2]


class GraderError(Exception):
    """Base class for safe grader errors."""

    code = "grader_error"
    http_status = HTTPStatus.INTERNAL_SERVER_ERROR


class InvalidSubmission(GraderError):
    code = "invalid_submission"
    http_status = HTTPStatus.BAD_REQUEST


class UnknownCase(GraderError):
    code = "unknown_case"
    http_status = HTTPStatus.NOT_FOUND


class AlreadyGraded(GraderError):
    code = "already_graded"
    http_status = HTTPStatus.CONFLICT


class GraderConfigurationError(GraderError):
    code = "grader_configuration_error"
    http_status = HTTPStatus.INTERNAL_SERVER_ERROR


def canonical_json(value: Any) -> bytes:
    return json.dumps(
        value,
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
        allow_nan=False,
    ).encode("utf-8")


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def reject_json_constant(value: str) -> None:
    raise ValueError(f"non-finite JSON number is not allowed: {value}")


def parse_json_bytes(value: bytes, description: str) -> Any:
    try:
        return json.loads(
            value.decode("utf-8"),
            parse_constant=reject_json_constant,
        )
    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
        raise InvalidSubmission(f"{description} is not valid strict JSON") from exc


def read_json_object(path: Path, description: str) -> dict[str, Any]:
    try:
        raw = path.read_bytes()
    except OSError as exc:
        raise GraderConfigurationError(f"cannot read {description}") from exc
    try:
        value = parse_json_bytes(raw, description)
    except InvalidSubmission as exc:
        raise GraderConfigurationError(f"{description} is invalid") from exc
    if not isinstance(value, dict):
        raise GraderConfigurationError(f"{description} must be a JSON object")
    return value


def required_identifier(value: Any, field: str) -> str:
    if not isinstance(value, str) or not IDENTIFIER_PATTERN.fullmatch(value):
        raise InvalidSubmission(
            f"{field} must match {IDENTIFIER_PATTERN.pattern}"
        )
    return value


def required_string(value: Any, field: str, maximum: int = 2048) -> str:
    if not isinstance(value, str) or not value.strip() or len(value) > maximum:
        raise InvalidSubmission(f"{field} must be a non-empty string")
    return value


def is_number(value: Any) -> bool:
    return isinstance(value, (int, float)) and not isinstance(value, bool)


def decimal_number(value: Any, field: str) -> Decimal:
    if not is_number(value):
        raise GraderConfigurationError(f"{field} must be numeric")
    if isinstance(value, float) and not math.isfinite(value):
        raise GraderConfigurationError(f"{field} must be finite")
    try:
        return Decimal(str(value))
    except InvalidOperation as exc:
        raise GraderConfigurationError(f"{field} must be numeric") from exc


def normalize_comparison(value: Any) -> dict[str, Any]:
    if isinstance(value, str):
        return {"type": value}
    if isinstance(value, dict):
        return value
    raise GraderConfigurationError("target comparison must be a string or object")


def compare_answer(actual: Any, expected: Any, comparison: Any) -> bool:
    rule = normalize_comparison(comparison)
    rule_type = rule.get("type")
    if rule_type == "exact":
        if is_number(actual) and is_number(expected):
            return decimal_number(actual, "actual") == decimal_number(expected, "expected")
        return type(actual) is type(expected) and actual == expected
    if rule_type == "case_insensitive_exact":
        if not isinstance(actual, str) or not isinstance(expected, str):
            return False
        return actual.casefold() == expected.casefold()
    if rule_type == "three_decimal_places":
        if not is_number(actual) or not is_number(expected):
            return False
        quantum = Decimal("0.001")
        return decimal_number(actual, "actual").quantize(
            quantum, rounding=ROUND_HALF_UP
        ) == decimal_number(expected, "expected").quantize(
            quantum, rounding=ROUND_HALF_UP
        )
    if rule_type == "absolute_tolerance":
        if not is_number(actual) or not is_number(expected):
            return False
        tolerance = decimal_number(rule.get("tolerance"), "comparison.tolerance")
        if tolerance < 0:
            raise GraderConfigurationError("comparison.tolerance cannot be negative")
        return abs(
            decimal_number(actual, "actual") - decimal_number(expected, "expected")
        ) <= tolerance
    raise GraderConfigurationError(f"unsupported comparison type: {rule_type!r}")


def validate_evidence(value: Any, answer_index: int) -> None:
    if not isinstance(value, list) or not value:
        raise InvalidSubmission(
            f"answers[{answer_index}].evidence must be a non-empty array"
        )
    for evidence_index, item in enumerate(value):
        prefix = f"answers[{answer_index}].evidence[{evidence_index}]"
        if not isinstance(item, dict):
            raise InvalidSubmission(f"{prefix} must be an object")
        required_string(item.get("artifact_uri"), f"{prefix}.artifact_uri")
        digest = item.get("artifact_sha256")
        if not isinstance(digest, str) or not SHA256_PATTERN.fullmatch(digest):
            raise InvalidSubmission(f"{prefix}.artifact_sha256 must be lowercase SHA-256")
        if "json_pointer" in item:
            pointer = item["json_pointer"]
            if not isinstance(pointer, str) or (pointer and not pointer.startswith("/")):
                raise InvalidSubmission(f"{prefix}.json_pointer must be a JSON Pointer")


def validate_submission(value: Any) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise InvalidSubmission("submission must be a JSON object")
    if value.get("schema_version") != SCHEMA_VERSION:
        raise InvalidSubmission(f"schema_version must equal {SCHEMA_VERSION}")
    required_identifier(value.get("submission_id"), "submission_id")
    required_identifier(value.get("case_id"), "case_id")
    required_identifier(value.get("run_id"), "run_id")
    if value.get("phase") != "final":
        raise InvalidSubmission("phase must equal final")
    answers = value.get("answers")
    if not isinstance(answers, list) or len(answers) > 512:
        raise InvalidSubmission("answers must be an array with at most 512 entries")
    seen: set[str] = set()
    for index, answer in enumerate(answers):
        if not isinstance(answer, dict):
            raise InvalidSubmission(f"answers[{index}] must be an object")
        target_id = required_identifier(answer.get("target_id"), f"answers[{index}].target_id")
        if target_id in seen:
            raise InvalidSubmission(f"duplicate target_id: {target_id}")
        seen.add(target_id)
        if "value" not in answer:
            raise InvalidSubmission(f"answers[{index}].value is required")
        validate_evidence(answer.get("evidence"), index)
    return value


def validate_key(value: dict[str, Any], expected_case_id: str) -> list[dict[str, Any]]:
    if value.get("schema_version") != SCHEMA_VERSION:
        raise GraderConfigurationError(f"answer-key schema_version must equal {SCHEMA_VERSION}")
    if value.get("visibility") != "grader_only":
        raise GraderConfigurationError("answer-key visibility must equal grader_only")
    if value.get("case_id") != expected_case_id:
        raise GraderConfigurationError("answer-key case_id does not match requested case")
    try:
        required_string(value.get("key_id"), "answer-key key_id", maximum=256)
    except InvalidSubmission as exc:
        raise GraderConfigurationError(str(exc)) from exc
    targets = value.get("targets")
    if not isinstance(targets, list) or not targets or len(targets) > 512:
        raise GraderConfigurationError("answer-key targets must contain 1 to 512 entries")
    seen: set[str] = set()
    normalized: list[dict[str, Any]] = []
    for index, target in enumerate(targets):
        if not isinstance(target, dict):
            raise GraderConfigurationError(f"answer-key targets[{index}] must be an object")
        try:
            target_id = required_identifier(target.get("id"), f"targets[{index}].id")
        except InvalidSubmission as exc:
            raise GraderConfigurationError(str(exc)) from exc
        if target_id in seen:
            raise GraderConfigurationError(f"duplicate answer-key target id: {target_id}")
        seen.add(target_id)
        if "expected" not in target:
            raise GraderConfigurationError(f"answer-key target {target_id} lacks expected")
        comparison = normalize_comparison(target.get("comparison"))
        points = target.get("points", 1)
        if not is_number(points) or points <= 0:
            raise GraderConfigurationError(f"answer-key target {target_id} points must be positive")
        normalized.append(
            {
                "id": target_id,
                "expected": target["expected"],
                "comparison": comparison,
                "points": Decimal(str(points)),
            }
        )
    return normalized


def display_decimal(value: Decimal) -> int | float:
    integral = value.to_integral_value()
    if value == integral:
        return int(integral)
    return float(value)


def atomic_write_json(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    descriptor, temporary_name = tempfile.mkstemp(
        prefix=f".{path.name}.", suffix=".tmp", dir=path.parent
    )
    temporary = Path(temporary_name)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "wb") as handle:
            handle.write(canonical_json(value) + b"\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink()


def within(path: Path, root: Path) -> bool:
    resolved = path.resolve()
    parent = root.resolve()
    return resolved == parent or parent in resolved.parents


def private_mode(path: Path) -> bool:
    return stat.S_IMODE(path.stat().st_mode) & 0o077 == 0


def validate_service_paths(
    key_root: Path,
    receipt_root: Path,
    token_file: Path,
    allow_insecure_development: bool,
) -> None:
    for path, label in (
        (key_root, "key root"),
        (receipt_root, "receipt root"),
    ):
        if not path.exists() or not path.is_dir():
            raise GraderConfigurationError(f"{label} must be an existing directory")
    if not token_file.exists() or not token_file.is_file():
        raise GraderConfigurationError("token file must be an existing file")
    if allow_insecure_development:
        return
    for path, label in (
        (key_root, "key root"),
        (receipt_root, "receipt root"),
        (token_file, "token file"),
    ):
        if within(path, PROJECT_ROOT):
            raise GraderConfigurationError(f"{label} cannot be inside the project workspace")
        if not private_mode(path):
            raise GraderConfigurationError(f"{label} must not be group/world accessible")


class GradeService:
    def __init__(self, key_root: Path, receipt_root: Path) -> None:
        self.key_root = key_root.resolve()
        self.receipt_root = receipt_root.resolve()
        self._lock = threading.Lock()

    def key_path(self, case_id: str) -> Path:
        required_identifier(case_id, "case_id")
        return self.key_root / f"{case_id}.json"

    def receipt_path(self, case_id: str, run_id: str) -> Path:
        required_identifier(case_id, "case_id")
        required_identifier(run_id, "run_id")
        return self.receipt_root / case_id / f"{run_id}.json"

    def load_key(self, case_id: str) -> tuple[dict[str, Any], list[dict[str, Any]], Path]:
        path = self.key_path(case_id)
        if not path.is_file() or path.is_symlink():
            raise UnknownCase("no grader key is registered for this case")
        if not within(path, self.key_root):
            raise GraderConfigurationError("answer key resolves outside the key root")
        if not private_mode(path):
            raise GraderConfigurationError("answer key must not be group/world accessible")
        key = read_json_object(path, "answer key")
        targets = validate_key(key, case_id)
        return key, targets, path

    def grade(self, supplied: Any) -> dict[str, Any]:
        submission = validate_submission(supplied)
        submission_bytes = canonical_json(submission)
        submission_sha256 = sha256_bytes(submission_bytes)
        case_id = submission["case_id"]
        run_id = submission["run_id"]
        receipt_path = self.receipt_path(case_id, run_id)

        with self._lock:
            if receipt_path.exists():
                existing = read_json_object(receipt_path, "grading receipt")
                internal = existing.get("_internal", {})
                if internal.get("submission_sha256") != submission_sha256:
                    raise AlreadyGraded(
                        "this case and run_id were already graded with a different submission"
                    )
                public = existing.get("public")
                if not isinstance(public, dict):
                    raise GraderConfigurationError("stored grading receipt is invalid")
                return public

            key, targets, key_path = self.load_key(case_id)
            answers = {answer["target_id"]: answer for answer in submission["answers"]}
            target_ids = {target["id"] for target in targets}
            unexpected = sorted(set(answers) - target_ids)
            if unexpected:
                raise InvalidSubmission(
                    "submission contains target IDs not registered for this case"
                )

            results: list[dict[str, Any]] = []
            possible = Decimal("0")
            awarded = Decimal("0")
            for target in targets:
                possible += target["points"]
                answer = answers.get(target["id"])
                if answer is None:
                    status = "missing"
                    passed = False
                else:
                    passed = compare_answer(
                        answer["value"], target["expected"], target["comparison"]
                    )
                    status = "passed" if passed else "failed"
                if passed:
                    awarded += target["points"]
                results.append(
                    {
                        "target_id": target["id"],
                        "status": status,
                        "comparison": target["comparison"]["type"],
                        "points_awarded": display_decimal(target["points"] if passed else Decimal("0")),
                        "points_possible": display_decimal(target["points"]),
                    }
                )

            receipt_id = sha256_bytes(
                canonical_json(
                    {
                        "case_id": case_id,
                        "run_id": run_id,
                        "submission_sha256": submission_sha256,
                    }
                )
            )[:24]
            public = {
                "schema_version": SCHEMA_VERSION,
                "service": SERVICE_NAME,
                "service_version": SERVICE_VERSION,
                "receipt_id": receipt_id,
                "submission_id": submission["submission_id"],
                "case_id": case_id,
                "run_id": run_id,
                "phase": "final",
                "graded_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
                "status": "passed" if awarded == possible else "failed",
                "score": {
                    "points_awarded": display_decimal(awarded),
                    "points_possible": display_decimal(possible),
                    "fraction": float(awarded / possible),
                },
                "targets": results,
                "expected_values_disclosed": False,
                "idempotent": True,
            }
            stored = {
                "public": public,
                "_internal": {
                    "submission_sha256": submission_sha256,
                    "answer_key_sha256": sha256_file(key_path),
                    "key_id": key["key_id"],
                },
            }
            atomic_write_json(receipt_path, stored)
            return public


def read_token(path: Path) -> str:
    try:
        token = path.read_text(encoding="utf-8").strip()
    except OSError as exc:
        raise GraderConfigurationError("cannot read token file") from exc
    if len(token) < 32:
        raise GraderConfigurationError("token must contain at least 32 characters")
    return token


def handler_factory(service: GradeService, token: str) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        server_version = f"{SERVICE_NAME}/{SERVICE_VERSION}"

        def log_message(self, format: str, *args: Any) -> None:
            return

        def send_json(self, status: HTTPStatus, value: dict[str, Any]) -> None:
            payload = canonical_json(value) + b"\n"
            self.send_response(status.value)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(payload)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(payload)

        def authenticated(self) -> bool:
            supplied = self.headers.get("Authorization", "")
            prefix = "Bearer "
            return supplied.startswith(prefix) and hmac.compare_digest(
                supplied[len(prefix) :], token
            )

        def do_GET(self) -> None:
            if self.path != "/health":
                self.send_json(HTTPStatus.NOT_FOUND, {"error": "not_found"})
                return
            self.send_json(
                HTTPStatus.OK,
                {
                    "status": "ok",
                    "service": SERVICE_NAME,
                    "version": SERVICE_VERSION,
                },
            )

        def do_POST(self) -> None:
            if self.path != "/v1/grade":
                self.send_json(HTTPStatus.NOT_FOUND, {"error": "not_found"})
                return
            if not self.authenticated():
                self.send_json(HTTPStatus.UNAUTHORIZED, {"error": "unauthorized"})
                return
            raw_length = self.headers.get("Content-Length")
            try:
                length = int(raw_length or "")
            except ValueError:
                length = -1
            if length < 0 or length > MAX_REQUEST_BYTES:
                self.send_json(
                    HTTPStatus.REQUEST_ENTITY_TOO_LARGE,
                    {"error": "invalid_content_length"},
                )
                return
            payload = self.rfile.read(length)
            try:
                submission = parse_json_bytes(payload, "request body")
                receipt = service.grade(submission)
            except GraderError as exc:
                self.send_json(
                    exc.http_status,
                    {"error": exc.code, "message": str(exc)},
                )
                return
            self.send_json(HTTPStatus.OK, receipt)

    return Handler


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    identify = subparsers.add_parser("identify", help="print the service contract")
    identify.set_defaults(command="identify")

    serve = subparsers.add_parser("serve", help="start the loopback grading service")
    serve.add_argument("--host", default="127.0.0.1")
    serve.add_argument("--port", type=int, default=8765)
    serve.add_argument("--key-root", type=Path, required=True)
    serve.add_argument("--receipt-root", type=Path, required=True)
    serve.add_argument("--token-file", type=Path, required=True)
    serve.add_argument(
        "--allow-insecure-development",
        action="store_true",
        help="allow project-local or broadly readable paths for development only",
    )
    return parser


def main() -> int:
    args = build_parser().parse_args()
    if args.command == "identify":
        print(
            json.dumps(
                {
                    "service": SERVICE_NAME,
                    "version": SERVICE_VERSION,
                    "protocol": SCHEMA_VERSION,
                    "transport": "HTTP loopback",
                    "endpoint": "POST /v1/grade",
                    "expected_values_disclosed": False,
                    "one_submission_per_case_run": True,
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0

    if args.host not in {"127.0.0.1", "::1", "localhost"}:
        raise SystemExit("Refusing to bind outside the loopback interface")
    if not (1 <= args.port <= 65535):
        raise SystemExit("port must be between 1 and 65535")
    try:
        validate_service_paths(
            args.key_root,
            args.receipt_root,
            args.token_file,
            args.allow_insecure_development,
        )
        token = read_token(args.token_file)
    except GraderConfigurationError as exc:
        raise SystemExit(str(exc)) from exc
    service = GradeService(args.key_root, args.receipt_root)
    server = ThreadingHTTPServer((args.host, args.port), handler_factory(service, token))
    print(
        json.dumps(
            {
                "status": "listening",
                "service": SERVICE_NAME,
                "version": SERVICE_VERSION,
                "host": args.host,
                "port": server.server_address[1],
            },
            sort_keys=True,
        ),
        flush=True,
    )
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
