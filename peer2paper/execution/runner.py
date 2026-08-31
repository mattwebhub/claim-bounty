#!/usr/bin/env python3
"""Peer2Paper's shared local scientific execution runner.

The runner deliberately owns only three operations:

* identify: describe this runner and the controls available on the host;
* preflight: validate one immutable execution manifest without running the study;
* run: execute that manifest once and emit a machine-readable attempt record.

The Toone routine runtime owns workflow attempts and retries. This runner never
retries a command or selects a repair candidate.
"""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import re
import resource
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from typing import Any


PROTOCOL_VERSION = "1.0.0"
DEFAULT_TIMEOUT_SECONDS = 600
DEFAULT_MAXIMUM_MEMORY_BYTES = 8 * 1024**3
DEFAULT_MAXIMUM_WORKING_STORAGE_BYTES = 5 * 1024**3
TYPOGRAPHIC_QUOTES = {"\u2018", "\u2019", "\u201c", "\u201d"}
FAILURE_CLASSES = {
    "missing_dependency",
    "invalid_path",
    "missing_file",
    "version_conflict",
    "permission_denied",
    "isolation_unavailable",
    "resource_control_unavailable",
    "runtime_error",
    "numerical_mismatch",
    "invalid_manifest",
    "invalid_wrapper",
}


def canonical_json(value: Any) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def path_identity(path: Path) -> dict[str, Any]:
    resolved = path.resolve()
    if resolved.is_file():
        return {"path": str(resolved), "kind": "file", "sha256": sha256_file(resolved)}
    if resolved.is_dir():
        digest = hashlib.sha256()
        file_count = 0
        for candidate in sorted(resolved.rglob("*"), key=lambda item: item.as_posix()):
            if not candidate.is_file() or candidate.is_symlink():
                continue
            relative = candidate.relative_to(resolved).as_posix()
            digest.update(relative.encode("utf-8"))
            digest.update(b"\0")
            digest.update(sha256_file(candidate).encode("ascii"))
            digest.update(b"\n")
            file_count += 1
        return {
            "path": str(resolved),
            "kind": "directory",
            "sha256": digest.hexdigest(),
            "fileCount": file_count,
        }
    return {"path": str(resolved), "kind": "missing", "sha256": None}


def effective_timeout_seconds(manifest: dict[str, Any]) -> int:
    return int(manifest.get("timeoutSeconds", DEFAULT_TIMEOUT_SECONDS))


def effective_limits(manifest: dict[str, Any]) -> dict[str, int]:
    supplied = manifest.get("limits")
    if not isinstance(supplied, dict):
        supplied = {}
    return {
        "maximumMemoryBytes": int(
            supplied.get("maximumMemoryBytes", DEFAULT_MAXIMUM_MEMORY_BYTES)
        ),
        "maximumWorkingStorageBytes": int(
            supplied.get(
                "maximumWorkingStorageBytes", DEFAULT_MAXIMUM_WORKING_STORAGE_BYTES
            )
        ),
    }


def verified_input_identities(manifest: dict[str, Any]) -> list[dict[str, Any]]:
    return [
        path_identity(Path(mount["path"]))
        for mount in manifest.get("mounts", [])
        if mount.get("verifyUnchanged") is True
    ]


def output_identities(attempt_directory: Path) -> list[dict[str, Any]]:
    identities: list[dict[str, Any]] = []
    for candidate in sorted(attempt_directory.rglob("*"), key=lambda item: item.as_posix()):
        if not candidate.is_file() or candidate.name == "attempt-record.json":
            continue
        identities.append(
            {
                "path": candidate.relative_to(attempt_directory).as_posix(),
                "sha256": sha256_file(candidate),
                "bytes": candidate.stat().st_size,
            }
        )
    return identities


def command_r_source(manifest: dict[str, Any]) -> Path | None:
    runtime = manifest.get("runtime")
    if not isinstance(runtime, dict) or runtime.get("kind") != "R":
        return None
    for argument in manifest.get("command", [])[1:]:
        if isinstance(argument, str) and argument.lower().endswith(".r"):
            return Path(argument).resolve()
    return None


def unsafe_r_source_failure(source: Path) -> dict[str, str] | None:
    try:
        content = source.read_text(encoding="utf-8")
    except OSError as exc:
        return {"class": "invalid_wrapper", "message": f"cannot read R command source: {exc}"}
    if any(character in content for character in TYPOGRAPHIC_QUOTES):
        return {
            "class": "invalid_wrapper",
            "message": "R command source contains a typographic quote; use an ASCII string literal encoder",
        }
    return None


def runner_identity() -> dict[str, Any]:
    runner_path = Path(__file__).resolve()
    sandbox = shutil.which("sandbox-exec")
    return {
        "protocolVersion": PROTOCOL_VERSION,
        "runnerPath": str(runner_path),
        "runnerSha256": sha256_file(runner_path),
        "host": {
            "system": platform.system(),
            "release": platform.release(),
            "machine": platform.machine(),
            "python": platform.python_version(),
        },
        "controls": {
            "networkIsolation": "sandbox-exec" if sandbox else "unavailable",
            "filesystemIsolation": "sandbox-exec-write-boundary" if sandbox else "unavailable",
            "memory": "operational-default-with-rlimit-and-rss-supervision",
            "workingStorage": "operational-default-with-attempt-directory-supervision",
            "attemptIsolation": "dedicated-directory",
            "defaultTimeoutSeconds": DEFAULT_TIMEOUT_SECONDS,
            "defaultMaximumMemoryBytes": DEFAULT_MAXIMUM_MEMORY_BYTES,
            "defaultMaximumWorkingStorageBytes": DEFAULT_MAXIMUM_WORKING_STORAGE_BYTES,
        },
    }


def load_manifest(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ValueError(f"Cannot read execution manifest: {exc}") from exc
    if not isinstance(value, dict):
        raise ValueError("Execution manifest must be a JSON object")
    return value


def required_string(manifest: dict[str, Any], key: str) -> str:
    value = manifest.get(key)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"{key} must be a non-empty string")
    return value


def validate_manifest(manifest: dict[str, Any]) -> list[dict[str, str]]:
    failures: list[dict[str, str]] = []
    try:
        if manifest.get("protocolVersion") != PROTOCOL_VERSION:
            raise ValueError(f"protocolVersion must equal {PROTOCOL_VERSION}")
        required_string(manifest, "runId")
        required_string(manifest, "attemptId")
        required_string(manifest, "attemptDirectory")
        required_string(manifest, "workingDirectory")
        command = manifest.get("command")
        if not isinstance(command, list) or not command or not all(
            isinstance(item, str) and item for item in command
        ):
            raise ValueError("command must be a non-empty array of strings")
        runtime = manifest.get("runtime")
        if not isinstance(runtime, dict) or runtime.get("kind") not in {"R", "Python"}:
            raise ValueError("runtime.kind must be R or Python")
        if manifest.get("network") not in {"disabled", "host-policy"}:
            raise ValueError("network must be disabled or host-policy")
        timeout_seconds = manifest.get("timeoutSeconds", DEFAULT_TIMEOUT_SECONDS)
        if not isinstance(timeout_seconds, int) or timeout_seconds <= 0:
            raise ValueError("timeoutSeconds must be a positive integer")
        limits = manifest.get("limits", {})
        if not isinstance(limits, dict):
            raise ValueError("limits must be an object when supplied")
        for key in ("maximumMemoryBytes", "maximumWorkingStorageBytes"):
            value = limits.get(
                key,
                DEFAULT_MAXIMUM_MEMORY_BYTES
                if key == "maximumMemoryBytes"
                else DEFAULT_MAXIMUM_WORKING_STORAGE_BYTES,
            )
            if not isinstance(value, int) or value <= 0:
                raise ValueError(f"limits.{key} must be a positive integer")
        attempt = Path(manifest["attemptDirectory"]).resolve()
        working = Path(manifest["workingDirectory"]).resolve()
        if working != attempt and attempt not in working.parents:
            raise ValueError("workingDirectory must be inside attemptDirectory")
        mounts = manifest.get("mounts", [])
        if not isinstance(mounts, list):
            raise ValueError("mounts must be an array")
        for mount in mounts:
            if not isinstance(mount, dict) or mount.get("mode") not in {"read-only", "read-write"}:
                raise ValueError("each mount must have path and read-only/read-write mode")
            required_string(mount, "path")
            if "verifyUnchanged" in mount and not isinstance(mount["verifyUnchanged"], bool):
                raise ValueError("mount.verifyUnchanged must be boolean when supplied")
            if mount["mode"] == "read-write":
                mounted = Path(mount["path"]).resolve()
                if mounted != attempt and attempt not in mounted.parents:
                    raise ValueError("read-write mounts must be inside attemptDirectory")
    except ValueError as exc:
        failures.append({"class": "invalid_manifest", "message": str(exc)})
    return failures


def runtime_version(command: str) -> tuple[str | None, str | None]:
    executable = shutil.which(command)
    if not executable:
        return None, None
    probes = [[executable, "--version"]]
    if Path(executable).name == "Rscript":
        probes = [[executable, "--version"]]
    for probe in probes:
        completed = subprocess.run(probe, capture_output=True, text=True, check=False)
        combined = (completed.stdout + completed.stderr).strip()
        if combined:
            return executable, combined.splitlines()[0]
    return executable, "version unavailable"


def version_tuple(value: str) -> tuple[int, ...]:
    match = re.search(r"\d+(?:\.\d+)+", value)
    return tuple(int(part) for part in match.group(0).split(".")) if match else ()


def package_probe(runtime: dict[str, Any]) -> tuple[list[str], dict[str, str]]:
    required = runtime.get("requiredPackages", [])
    if not isinstance(required, list) or not all(isinstance(item, str) for item in required):
        return ["runtime.requiredPackages must be an array of strings"], {}
    versions: dict[str, str] = {}
    missing: list[str] = []
    if runtime["kind"] == "Python":
        for package in required:
            spec = importlib.util.find_spec(package)
            if spec is None:
                missing.append(package)
            else:
                versions[package] = spec.origin or "present"
        return missing, versions

    rscript = runtime.get("executable", "Rscript")
    library = runtime.get("libraryPath")
    environment = os.environ.copy()
    if isinstance(library, str) and library:
        environment["R_LIBS_USER"] = library
    for package in required:
        expression = (
            f"p <- {json.dumps(package)}; "
            "if (!requireNamespace(p, quietly=TRUE)) quit(status=42); "
            "cat(p, as.character(packageVersion(p)), sep='=', fill=TRUE)"
        )
        completed = subprocess.run(
            [rscript, "-e", expression],
            capture_output=True,
            text=True,
            check=False,
            env=environment,
        )
        if completed.returncode != 0:
            missing.append(package)
        else:
            line = completed.stdout.strip().splitlines()[-1]
            versions[package] = line.split("=", 1)[-1] if "=" in line else "present"
    return missing, versions


def parse_r_command_source(
    manifest: dict[str, Any],
    executable: str,
    working_directory: Path,
    attempt_directory: Path,
) -> tuple[dict[str, Any] | None, dict[str, str] | None]:
    source = command_r_source(manifest)
    if source is None:
        return None, None
    if not source.is_file():
        failure = {
            "class": "invalid_wrapper",
            "message": f"R command source is missing: {source}",
        }
        return {"source": str(source), "succeeded": False}, failure
    unsafe = unsafe_r_source_failure(source)
    if unsafe:
        return {"source": str(source), "succeeded": False}, unsafe
    expression = f"parse(file={json.dumps(str(source))}, keep.source=TRUE); cat('runner-r-parse-ok\\n')"
    parse_manifest = dict(manifest)
    parse_manifest["command"] = [executable, "--vanilla", "-e", expression]
    try:
        completed = subprocess.run(
            sandbox_command(parse_manifest, attempt_directory),
            cwd=working_directory,
            env={**os.environ, "TMPDIR": str(attempt_directory)},
            capture_output=True,
            text=True,
            check=False,
            timeout=min(effective_timeout_seconds(manifest), 60),
        )
    except subprocess.TimeoutExpired:
        failure = {
            "class": "invalid_wrapper",
            "message": "R command source parse check exceeded 60 seconds",
        }
        return {"source": str(source), "succeeded": False}, failure
    check = {
        "source": str(source),
        "sourceSha256": sha256_file(source),
        "exitCode": completed.returncode,
        "succeeded": completed.returncode == 0,
        "stderr": completed.stderr[-2000:],
    }
    if completed.returncode != 0:
        return check, {
            "class": "invalid_wrapper",
            "message": "R command source failed parse-only validation",
        }
    return check, None


def preflight(manifest: dict[str, Any]) -> dict[str, Any]:
    failures = validate_manifest(manifest)
    identity = runner_identity()
    checks: dict[str, Any] = {"runner": identity}
    if failures:
        return {"status": "blocked", "checks": checks, "failures": failures}

    working_directory = Path(manifest["workingDirectory"]).resolve()
    attempt_directory = Path(manifest["attemptDirectory"]).resolve()
    if not working_directory.is_dir():
        failures.append({"class": "invalid_path", "message": "workingDirectory is not a directory"})
    elif not os.access(working_directory, os.R_OK | os.X_OK):
        failures.append({"class": "permission_denied", "message": "workingDirectory is not readable"})

    for mount in manifest.get("mounts", []):
        mount_path = Path(mount["path"]).resolve()
        if not mount_path.exists():
            failures.append({"class": "missing_file", "message": f"mount is missing: {mount_path}"})
        elif not os.access(mount_path, os.R_OK):
            failures.append({"class": "permission_denied", "message": f"mount is unreadable: {mount_path}"})
        elif mount["mode"] == "read-write" and not os.access(mount_path, os.W_OK):
            failures.append({"class": "permission_denied", "message": f"mount is not writable: {mount_path}"})

    command_name = manifest["command"][0]
    executable, version = runtime_version(command_name)
    checks["runtime"] = {"executable": executable, "version": version}
    if not executable:
        failures.append({"class": "missing_dependency", "message": f"runtime is unavailable: {command_name}"})
    else:
        minimum_version = manifest["runtime"].get("minimumVersion")
        if isinstance(minimum_version, str) and version:
            installed = version_tuple(version)
            required = version_tuple(minimum_version)
            if not installed or not required or installed < required:
                failures.append(
                    {
                        "class": "version_conflict",
                        "message": f"runtime {version} does not satisfy minimum {minimum_version}",
                    }
                )
        missing, versions = package_probe(manifest["runtime"])
        checks["packages"] = versions
        if missing:
            failures.append({"class": "missing_dependency", "message": "missing packages: " + ", ".join(missing)})

    if manifest["network"] == "disabled" and identity["controls"]["networkIsolation"] == "unavailable":
        failures.append({"class": "isolation_unavailable", "message": "sandbox-exec is required for disabled network"})
    if identity["controls"]["filesystemIsolation"] == "unavailable":
        failures.append(
            {
                "class": "isolation_unavailable",
                "message": "sandbox-exec is required to deny writes outside attemptDirectory",
            }
        )

    try:
        attempt_directory.mkdir(parents=True, exist_ok=True)
        with tempfile.NamedTemporaryFile(dir=attempt_directory, prefix="write-check-", delete=True):
            pass
        checks["attemptDirectoryWritable"] = True
    except OSError as exc:
        failures.append({"class": "permission_denied", "message": f"attempt directory is not writable: {exc}"})

    if executable and not failures and manifest["runtime"]["kind"] == "R":
        parse_check, parse_failure = parse_r_command_source(
            manifest, executable, working_directory, attempt_directory
        )
        if parse_check is not None:
            checks["rCommandSourceParse"] = parse_check
        if parse_failure is not None:
            failures.append(parse_failure)

    if executable and not failures:
        probe_command = (
            [executable, "-e", "cat('runner-preflight-ok\\n')"]
            if manifest["runtime"]["kind"] == "R"
            else [executable, "-c", "print('runner-preflight-ok')"]
        )
        probe_manifest = dict(manifest)
        probe_manifest["command"] = probe_command
        completed = subprocess.run(
            sandbox_command(probe_manifest, attempt_directory),
            cwd=working_directory,
            env={**os.environ, "TMPDIR": str(attempt_directory)},
            capture_output=True,
            text=True,
            check=False,
        )
        checks["trivialCommand"] = {
            "command": probe_command,
            "exitCode": completed.returncode,
            "succeeded": completed.returncode == 0,
        }
        if completed.returncode != 0:
            failures.append({"class": "runtime_error", "message": "trivial runtime command failed"})

    if manifest["network"] == "disabled" and identity["controls"]["networkIsolation"] != "unavailable":
        network_probe = [
            sys.executable,
            "-c",
            (
                "import errno,socket,sys; s=socket.socket(); "
                "r=s.connect_ex(('127.0.0.1',9)); "
                "sys.exit(0 if r in (errno.EPERM,errno.EACCES) else 3)"
            ),
        ]
        network_manifest = dict(manifest)
        network_manifest["command"] = network_probe
        completed = subprocess.run(
            sandbox_command(network_manifest, attempt_directory),
            cwd=working_directory if working_directory.is_dir() else None,
            env={**os.environ, "TMPDIR": str(attempt_directory)},
            capture_output=True,
            text=True,
            check=False,
        )
        checks["networkIsolationVerified"] = completed.returncode == 0
        if completed.returncode != 0:
            failures.append(
                {
                    "class": "isolation_unavailable",
                    "message": "network-denial probe did not return an enforced permission error",
                }
            )

    checks["manifestSha256"] = hashlib.sha256(canonical_json(manifest)).hexdigest()
    checks["environmentSha256"] = hashlib.sha256(
        canonical_json({"runtime": checks.get("runtime"), "packages": checks.get("packages", {})})
    ).hexdigest()
    checks["effectiveTimeoutSeconds"] = effective_timeout_seconds(manifest)
    checks["effectiveLimits"] = effective_limits(manifest)
    checks["inputIdentities"] = verified_input_identities(manifest)
    return {"status": "ready" if not failures else "blocked", "checks": checks, "failures": failures}


def directory_size(path: Path) -> int:
    total = 0
    for candidate in path.rglob("*"):
        try:
            if candidate.is_file() and not candidate.is_symlink():
                total += candidate.stat().st_size
        except OSError:
            continue
    return total


def resident_bytes(pid: int) -> int | None:
    completed = subprocess.run(
        ["ps", "-o", "rss=", "-p", str(pid)], capture_output=True, text=True, check=False
    )
    try:
        return int(completed.stdout.strip()) * 1024
    except ValueError:
        return None


def sandbox_command(manifest: dict[str, Any], attempt_directory: Path) -> list[str]:
    command = list(manifest["command"])
    if platform.system() != "Darwin" or not shutil.which("sandbox-exec"):
        return command
    profile = attempt_directory / "network-deny.sb"
    rules = [
        "(version 1)",
        "(allow default)",
        "(deny file-write*)",
        f"(allow file-write* (subpath {json.dumps(str(attempt_directory))}))",
    ]
    if manifest["network"] == "disabled":
        rules.append("(deny network*)")
    profile.write_text("\n".join(rules) + "\n", encoding="utf-8")
    return ["sandbox-exec", "-f", str(profile), *command]


def execute(manifest: dict[str, Any]) -> dict[str, Any]:
    readiness = preflight(manifest)
    if readiness["status"] != "ready":
        return {"status": "blocked", "preflight": readiness, "failureClass": readiness["failures"][0]["class"]}

    attempt_directory = Path(manifest["attemptDirectory"]).resolve()
    working_directory = Path(manifest["workingDirectory"]).resolve()
    stdout_path = attempt_directory / "stdout.log"
    stderr_path = attempt_directory / "stderr.log"
    limits = effective_limits(manifest)
    timeout_seconds = effective_timeout_seconds(manifest)
    maximum_memory = limits["maximumMemoryBytes"]
    maximum_storage = limits["maximumWorkingStorageBytes"]
    command = sandbox_command(manifest, attempt_directory)
    environment = os.environ.copy()
    environment.update({str(k): str(v) for k, v in manifest.get("environment", {}).items()})
    environment["TMPDIR"] = str(attempt_directory)
    if manifest["runtime"]["kind"] == "R":
        environment.setdefault("R_MAX_VSIZE", str(maximum_memory))

    def set_limits() -> None:
        os.setsid()
        try:
            resource.setrlimit(resource.RLIMIT_AS, (maximum_memory, maximum_memory))
        except (ValueError, OSError):
            pass

    started = time.time()
    monotonic_started = time.monotonic()
    termination: str | None = None
    with stdout_path.open("wb") as stdout, stderr_path.open("wb") as stderr:
        process = subprocess.Popen(
            command,
            cwd=working_directory,
            env=environment,
            stdout=stdout,
            stderr=stderr,
            preexec_fn=set_limits,
        )
        peak_resident = 0
        peak_storage = 0
        while process.poll() is None:
            current_memory = resident_bytes(process.pid)
            if current_memory is not None:
                peak_resident = max(peak_resident, current_memory)
                if current_memory > maximum_memory:
                    termination = "memory_limit_exceeded"
            current_storage = directory_size(attempt_directory)
            peak_storage = max(peak_storage, current_storage)
            if current_storage > maximum_storage:
                termination = "working_storage_limit_exceeded"
            if time.monotonic() - monotonic_started > timeout_seconds:
                termination = "timeout_exceeded"
            if termination:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                break
            time.sleep(0.25)
        return_code = process.wait()

    finished = time.time()
    input_identities_before = readiness["checks"].get("inputIdentities", [])
    input_identities_after = verified_input_identities(manifest)
    source_inputs_unchanged = input_identities_before == input_identities_after
    if not source_inputs_unchanged:
        termination = "source_modified"
    completed = return_code == 0 and termination is None
    record = {
        "status": "completed" if completed else "failed",
        "failureClass": None if completed else "runtime_error",
        "terminationReason": termination,
        "command": manifest["command"],
        "commandSha256": hashlib.sha256(canonical_json(manifest["command"])).hexdigest(),
        "executedCommand": command,
        "workingDirectory": str(working_directory),
        "attemptDirectory": str(attempt_directory),
        "runner": readiness["checks"]["runner"],
        "manifestSha256": readiness["checks"]["manifestSha256"],
        "environmentSha256": readiness["checks"]["environmentSha256"],
        "exitCode": return_code,
        "startedAtUnix": started,
        "finishedAtUnix": finished,
        "durationSeconds": round(finished - started, 3),
        "timeoutSeconds": timeout_seconds,
        "effectiveLimits": limits,
        "peakResidentBytes": peak_resident,
        "peakWorkingStorageBytes": peak_storage,
        "inputIdentitiesBefore": input_identities_before,
        "inputIdentitiesAfter": input_identities_after,
        "sourceInputsUnchanged": source_inputs_unchanged,
        "stdout": str(stdout_path),
        "stderr": str(stderr_path),
    }
    record["outputIdentities"] = output_identities(attempt_directory)
    record_path = attempt_directory / "attempt-record.json"
    record_path.write_text(json.dumps(record, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return record


def emit(value: dict[str, Any], output: Path | None) -> None:
    rendered = json.dumps(value, indent=2, sort_keys=True) + "\n"
    if output:
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(rendered, encoding="utf-8")
    sys.stdout.write(rendered)


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="operation", required=True)
    identify_parser = subparsers.add_parser("identify")
    identify_parser.add_argument("--output", type=Path)
    for operation in ("preflight", "run"):
        operation_parser = subparsers.add_parser(operation)
        operation_parser.add_argument("--manifest", required=True, type=Path)
        operation_parser.add_argument("--output", type=Path)
    args = parser.parse_args()

    if args.operation == "identify":
        result = runner_identity()
    else:
        try:
            manifest = load_manifest(args.manifest)
            result = preflight(manifest) if args.operation == "preflight" else execute(manifest)
        except ValueError as exc:
            result = {"status": "blocked", "failureClass": "invalid_manifest", "message": str(exc)}
    emit(result, args.output)
    return 0 if result.get("status") in {None, "ready", "completed"} else 2


if __name__ == "__main__":
    raise SystemExit(main())
