#!/usr/bin/env python3
"""Finalize and grade one terminated Peer2Paper validation run.

This command is the local bridge between sealed scientific artefacts and the
score-only grader. It has no answer-key access. It validates stage observation
sidecars, verifies their evidence hashes and JSON Pointers, seals one final
submission, and optionally sends it to the loopback grading service.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import re
import stat
import tempfile
from typing import Any
from urllib import error, parse, request


SCHEMA_VERSION = "1.0.0"
IDENTIFIER_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
PROJECT_ROOT = Path(__file__).resolve().parents[2]
STAGE_DIRECTORIES = {
    "study_case": "study-case",
    "reproduction": "reproduction",
    "robustness": "robustness",
    "research": "research",
    "verification": "verification",
    "delivery": "delivery",
}
STAGE_SIDECAR_NAME = "validation-observations.json"


class FinalizationError(Exception):
    """A safe error that prevents submission or overwrite."""


def reject_json_constant(value: str) -> None:
    raise ValueError(f"non-finite JSON number is not allowed: {value}")


def parse_json_bytes(value: bytes, description: str) -> Any:
    try:
        return json.loads(
            value.decode("utf-8"),
            parse_constant=reject_json_constant,
        )
    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
        raise FinalizationError(f"{description} is not valid strict JSON") from exc


def read_json_object(path: Path, description: str) -> dict[str, Any]:
    try:
        raw = path.read_bytes()
    except OSError as exc:
        raise FinalizationError(f"cannot read {description}: {path}") from exc
    value = parse_json_bytes(raw, description)
    if not isinstance(value, dict):
        raise FinalizationError(f"{description} must be a JSON object")
    return value


def canonical_json(value: Any) -> bytes:
    try:
        return json.dumps(
            value,
            sort_keys=True,
            separators=(",", ":"),
            ensure_ascii=False,
            allow_nan=False,
        ).encode("utf-8")
    except (TypeError, ValueError) as exc:
        raise FinalizationError("value cannot be encoded as strict JSON") from exc


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    try:
        with path.open("rb") as handle:
            for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                digest.update(chunk)
    except OSError as exc:
        raise FinalizationError(f"cannot hash evidence artefact: {path}") from exc
    return digest.hexdigest()


def within(path: Path, root: Path) -> bool:
    resolved = path.resolve()
    parent = root.resolve()
    return resolved == parent or parent in resolved.parents


def required_identifier(value: Any, field: str) -> str:
    if not isinstance(value, str) or not IDENTIFIER_PATTERN.fullmatch(value):
        raise FinalizationError(f"{field} must match {IDENTIFIER_PATTERN.pattern}")
    return value


def required_string(value: Any, field: str, maximum: int = 4096) -> str:
    if not isinstance(value, str) or not value.strip() or len(value) > maximum:
        raise FinalizationError(f"{field} must be a non-empty string")
    return value


def reject_unknown_keys(value: dict[str, Any], allowed: set[str], field: str) -> None:
    unexpected = sorted(set(value) - allowed)
    if unexpected:
        raise FinalizationError(
            f"{field} contains unsupported fields: {', '.join(unexpected)}"
        )


def validate_answer_type(value: Any, answer_type: str, target_id: str) -> None:
    valid = False
    if answer_type == "integer":
        valid = isinstance(value, int) and not isinstance(value, bool)
    elif answer_type == "number":
        valid = isinstance(value, (int, float)) and not isinstance(value, bool)
        if isinstance(value, float):
            valid = valid and math.isfinite(value)
    elif answer_type == "string":
        valid = isinstance(value, str)
    elif answer_type == "boolean":
        valid = isinstance(value, bool)
    if not valid:
        raise FinalizationError(
            f"observation {target_id} does not match answer_type {answer_type}"
        )


def validate_question_contract(value: Any) -> tuple[str, list[dict[str, Any]]]:
    if not isinstance(value, dict):
        raise FinalizationError("validation question contract must be an object")
    reject_unknown_keys(value, {"schema_version", "case_id", "targets"}, "question contract")
    if value.get("schema_version") != SCHEMA_VERSION:
        raise FinalizationError(f"question contract schema_version must equal {SCHEMA_VERSION}")
    case_id = required_identifier(value.get("case_id"), "question contract case_id")
    targets = value.get("targets")
    if not isinstance(targets, list) or not targets or len(targets) > 512:
        raise FinalizationError("question contract targets must contain 1 to 512 entries")
    seen: set[str] = set()
    normalized: list[dict[str, Any]] = []
    for index, target in enumerate(targets):
        prefix = f"question contract targets[{index}]"
        if not isinstance(target, dict):
            raise FinalizationError(f"{prefix} must be an object")
        reject_unknown_keys(
            target,
            {"id", "question", "answer_type", "producer_stage", "evidence_requirement"},
            prefix,
        )
        target_id = required_identifier(target.get("id"), f"{prefix}.id")
        if target_id in seen:
            raise FinalizationError(f"duplicate question target id: {target_id}")
        seen.add(target_id)
        required_string(target.get("question"), f"{prefix}.question")
        required_string(
            target.get("evidence_requirement"), f"{prefix}.evidence_requirement"
        )
        answer_type = target.get("answer_type")
        if answer_type not in {"integer", "number", "string", "boolean"}:
            raise FinalizationError(f"{prefix}.answer_type is unsupported")
        producer_stage = target.get("producer_stage")
        if producer_stage not in STAGE_DIRECTORIES:
            raise FinalizationError(f"{prefix}.producer_stage is unsupported")
        normalized.append(
            {
                "id": target_id,
                "answer_type": answer_type,
                "producer_stage": producer_stage,
            }
        )
    return case_id, normalized


def resolve_json_pointer(document: Any, pointer: str) -> Any:
    if pointer == "":
        return document
    if not pointer.startswith("/"):
        raise FinalizationError("JSON Pointer must be empty or begin with /")
    current = document
    for raw_token in pointer[1:].split("/"):
        token = raw_token.replace("~1", "/").replace("~0", "~")
        if isinstance(current, dict):
            if token not in current:
                raise FinalizationError(f"JSON Pointer member does not exist: {pointer}")
            current = current[token]
        elif isinstance(current, list):
            if not token.isdigit():
                raise FinalizationError(f"JSON Pointer array index is invalid: {pointer}")
            index = int(token)
            if index >= len(current):
                raise FinalizationError(f"JSON Pointer array index is out of range: {pointer}")
            current = current[index]
        else:
            raise FinalizationError(f"JSON Pointer traverses a scalar value: {pointer}")
    return current


def artifact_path_for_uri(artifact_uri: str, run_id: str, run_root: Path) -> Path:
    prefix = f"project://peer2paper/audits/{run_id}/"
    if not artifact_uri.startswith(prefix):
        raise FinalizationError(
            f"evidence artifact_uri must begin with {prefix}"
        )
    relative = artifact_uri[len(prefix) :]
    if not relative:
        raise FinalizationError("evidence artifact_uri must identify a file")
    candidate = (run_root / relative).resolve()
    if not within(candidate, run_root):
        raise FinalizationError("evidence artifact_uri resolves outside the run root")
    if not candidate.is_file() or candidate.is_symlink():
        raise FinalizationError(f"evidence artefact is not a regular file: {artifact_uri}")
    return candidate


def validate_evidence(
    value: Any,
    target_id: str,
    observed_value: Any,
    run_id: str,
    run_root: Path,
) -> list[dict[str, str]]:
    if not isinstance(value, list) or not value:
        raise FinalizationError(f"observation {target_id} evidence must be non-empty")
    normalized: list[dict[str, str]] = []
    for index, evidence in enumerate(value):
        prefix = f"observation {target_id} evidence[{index}]"
        if not isinstance(evidence, dict):
            raise FinalizationError(f"{prefix} must be an object")
        reject_unknown_keys(
            evidence,
            {"artifact_uri", "artifact_sha256", "json_pointer"},
            prefix,
        )
        artifact_uri = required_string(evidence.get("artifact_uri"), f"{prefix}.artifact_uri")
        supplied_hash = evidence.get("artifact_sha256")
        if not isinstance(supplied_hash, str) or not SHA256_PATTERN.fullmatch(supplied_hash):
            raise FinalizationError(f"{prefix}.artifact_sha256 must be lowercase SHA-256")
        pointer = evidence.get("json_pointer")
        if not isinstance(pointer, str):
            raise FinalizationError(f"{prefix}.json_pointer is required for JSON evidence")
        artifact_path = artifact_path_for_uri(artifact_uri, run_id, run_root)
        actual_hash = sha256_file(artifact_path)
        if actual_hash != supplied_hash:
            raise FinalizationError(f"evidence hash mismatch for {artifact_uri}")
        document = read_json_object(artifact_path, f"evidence artefact {artifact_uri}")
        pointed_value = resolve_json_pointer(document, pointer)
        if canonical_json(pointed_value) != canonical_json(observed_value):
            raise FinalizationError(
                f"evidence JSON Pointer value does not match observation {target_id}"
            )
        normalized.append(
            {
                "artifact_uri": artifact_uri,
                "artifact_sha256": supplied_hash,
                "json_pointer": pointer,
            }
        )
    return normalized


def collect_stage_observations(
    run_root: Path,
    run_id: str,
    case_id: str,
    targets: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    target_map = {target["id"]: target for target in targets}
    collected: dict[str, dict[str, Any]] = {}
    for stage, directory in STAGE_DIRECTORIES.items():
        sidecar_path = run_root / directory / STAGE_SIDECAR_NAME
        if not sidecar_path.exists():
            continue
        if not sidecar_path.is_file() or sidecar_path.is_symlink():
            raise FinalizationError(f"stage observation sidecar is not a regular file: {sidecar_path}")
        sidecar = read_json_object(sidecar_path, f"{stage} observation sidecar")
        reject_unknown_keys(
            sidecar,
            {"schema_version", "case_id", "run_id", "producer_stage", "observations"},
            f"{stage} observation sidecar",
        )
        if sidecar.get("schema_version") != SCHEMA_VERSION:
            raise FinalizationError(f"{stage} sidecar schema_version must equal {SCHEMA_VERSION}")
        if sidecar.get("case_id") != case_id:
            raise FinalizationError(f"{stage} sidecar case_id does not match the contract")
        if sidecar.get("run_id") != run_id:
            raise FinalizationError(f"{stage} sidecar run_id does not match the run directory")
        if sidecar.get("producer_stage") != stage:
            raise FinalizationError(f"{stage} sidecar producer_stage does not match its directory")
        observations = sidecar.get("observations")
        if not isinstance(observations, list) or len(observations) > 512:
            raise FinalizationError(f"{stage} sidecar observations must be an array")
        for index, observation in enumerate(observations):
            prefix = f"{stage} observations[{index}]"
            if not isinstance(observation, dict):
                raise FinalizationError(f"{prefix} must be an object")
            reject_unknown_keys(
                observation,
                {"target_id", "value", "evidence"},
                prefix,
            )
            target_id = required_identifier(observation.get("target_id"), f"{prefix}.target_id")
            if target_id in collected:
                raise FinalizationError(f"duplicate observation target_id: {target_id}")
            target = target_map.get(target_id)
            if target is None:
                raise FinalizationError(f"observation target is not in the visible contract: {target_id}")
            if target["producer_stage"] != stage:
                raise FinalizationError(
                    f"observation {target_id} came from {stage}, expected {target['producer_stage']}"
                )
            if "value" not in observation:
                raise FinalizationError(f"{prefix}.value is required")
            observed_value = observation["value"]
            validate_answer_type(observed_value, target["answer_type"], target_id)
            evidence = validate_evidence(
                observation.get("evidence"),
                target_id,
                observed_value,
                run_id,
                run_root,
            )
            collected[target_id] = {
                "target_id": target_id,
                "value": observed_value,
                "producer_stage": stage,
                "evidence": evidence,
            }
    return [collected[target["id"]] for target in targets if target["id"] in collected]


def sealed_write_json(path: Path, value: dict[str, Any]) -> None:
    payload = canonical_json(value) + b"\n"
    if path.exists():
        try:
            existing = path.read_bytes()
        except OSError as exc:
            raise FinalizationError(f"cannot read existing sealed output: {path}") from exc
        try:
            existing_value = parse_json_bytes(existing, f"existing sealed output {path.name}")
        except FinalizationError:
            raise
        if canonical_json(existing_value) != canonical_json(value):
            raise FinalizationError(f"refusing to overwrite changed sealed output: {path}")
        return
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    descriptor, temporary_name = tempfile.mkstemp(
        prefix=f".{path.name}.", suffix=".tmp", dir=path.parent
    )
    temporary = Path(temporary_name)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "wb") as handle:
            handle.write(payload)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink()


def read_token(path: Path) -> str:
    if not path.is_file() or path.is_symlink():
        raise FinalizationError("token file must be a regular file")
    if within(path, PROJECT_ROOT):
        raise FinalizationError("token file cannot be inside the project workspace")
    if stat.S_IMODE(path.stat().st_mode) & 0o077:
        raise FinalizationError("token file must not be group/world accessible")
    try:
        token = path.read_text(encoding="utf-8").strip()
    except OSError as exc:
        raise FinalizationError("cannot read grader token file") from exc
    if len(token) < 32:
        raise FinalizationError("grader token must contain at least 32 characters")
    return token


def validate_grader_url(value: str) -> str:
    parsed = parse.urlparse(value)
    if parsed.scheme != "http" or parsed.hostname not in {"127.0.0.1", "::1", "localhost"}:
        raise FinalizationError("grader URL must use HTTP on the loopback interface")
    if parsed.path != "/v1/grade" or parsed.params or parsed.query or parsed.fragment:
        raise FinalizationError("grader URL must identify /v1/grade without parameters")
    return value


def submit_to_grader(
    submission: dict[str, Any], grader_url: str, token_file: Path
) -> dict[str, Any]:
    endpoint = validate_grader_url(grader_url)
    token = read_token(token_file)
    outbound = request.Request(
        endpoint,
        data=canonical_json(submission),
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    try:
        with request.urlopen(outbound, timeout=10) as response:
            body = response.read()
    except error.HTTPError as exc:
        body = exc.read()
        try:
            detail = parse_json_bytes(body, "grader error response")
        except FinalizationError:
            detail = {"error": "invalid_grader_response"}
        message = detail.get("message") if isinstance(detail, dict) else None
        safe_message = message if isinstance(message, str) else "grader rejected the submission"
        raise FinalizationError(f"grader HTTP {exc.code}: {safe_message}") from exc
    except OSError as exc:
        raise FinalizationError("cannot connect to the loopback grader") from exc
    receipt = parse_json_bytes(body, "grader response")
    if not isinstance(receipt, dict):
        raise FinalizationError("grader response must be a JSON object")
    if receipt.get("expected_values_disclosed") is not False:
        raise FinalizationError("grader response failed the disclosure boundary")
    return receipt


def finalize_run(
    run_root: Path,
    question_contract_path: Path,
    terminal_outcome: str,
    grader_url: str | None = None,
    token_file: Path | None = None,
) -> dict[str, Any]:
    run_root = run_root.resolve()
    if not run_root.is_dir():
        raise FinalizationError("run root must be an existing directory")
    run_id = required_identifier(run_root.name, "run_id")
    terminal_outcome = required_identifier(terminal_outcome, "terminal_outcome")
    contract = read_json_object(question_contract_path, "validation question contract")
    case_id, targets = validate_question_contract(contract)
    observations = collect_stage_observations(run_root, run_id, case_id, targets)
    observed_ids = {item["target_id"] for item in observations}
    missing_target_ids = [target["id"] for target in targets if target["id"] not in observed_ids]
    finalized_observations = {
        "schema_version": SCHEMA_VERSION,
        "case_id": case_id,
        "run_id": run_id,
        "terminal_outcome": terminal_outcome,
        "observations": observations,
        "missing_target_ids": missing_target_ids,
    }
    submission_id = "final-" + sha256_bytes(
        canonical_json({"case_id": case_id, "run_id": run_id})
    )[:32]
    submission = {
        "schema_version": SCHEMA_VERSION,
        "submission_id": submission_id,
        "case_id": case_id,
        "run_id": run_id,
        "phase": "final",
        "answers": [
            {
                "target_id": item["target_id"],
                "value": item["value"],
                "evidence": item["evidence"],
            }
            for item in observations
        ],
    }
    output_root = run_root / "validation"
    observations_path = output_root / "observations.json"
    submission_path = output_root / "grading-submission.json"
    sealed_write_json(observations_path, finalized_observations)
    sealed_write_json(submission_path, submission)
    result: dict[str, Any] = {
        "status": "finalized",
        "case_id": case_id,
        "run_id": run_id,
        "terminal_outcome": terminal_outcome,
        "observation_count": len(observations),
        "missing_target_ids": missing_target_ids,
        "observations_path": str(observations_path),
        "submission_path": str(submission_path),
        "receipt_path": None,
    }
    if grader_url is None and token_file is None:
        return result
    if grader_url is None or token_file is None:
        raise FinalizationError("grader URL and token file must be supplied together")
    receipt = submit_to_grader(submission, grader_url, token_file)
    if receipt.get("case_id") != case_id or receipt.get("run_id") != run_id:
        raise FinalizationError("grader receipt identity does not match the finalized run")
    receipt_path = output_root / "grading-receipt.json"
    sealed_write_json(receipt_path, receipt)
    result["status"] = "graded"
    result["receipt_path"] = str(receipt_path)
    result["score"] = receipt.get("score")
    return result


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run-root", type=Path, required=True)
    parser.add_argument("--question-contract", type=Path, required=True)
    parser.add_argument("--terminal-outcome", required=True)
    parser.add_argument(
        "--grader-url",
        default=None,
        help="loopback POST /v1/grade URL; omit with --token-file for finalize-only mode",
    )
    parser.add_argument(
        "--token-file",
        type=Path,
        default=None,
        help="private bearer-token file; omit with --grader-url for finalize-only mode",
    )
    return parser


def main() -> int:
    args = build_parser().parse_args()
    try:
        result = finalize_run(
            run_root=args.run_root,
            question_contract_path=args.question_contract,
            terminal_outcome=args.terminal_outcome,
            grader_url=args.grader_url,
            token_file=args.token_file,
        )
    except FinalizationError as exc:
        raise SystemExit(str(exc)) from exc
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
