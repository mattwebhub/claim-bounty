#!/usr/bin/env python3
"""ClaimBounty's shared local scientific execution runner.

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
import json
import os
from pathlib import Path
import platform
import re
import resource
import secrets
import shutil
import signal
import stat
import subprocess
import sys
import tempfile
import time
from typing import Any


PROTOCOL_VERSION = "1.2.0"
EXECUTION_POLICY_SCHEMA_VERSION = "1.2.0"
DEFAULT_TIMEOUT_SECONDS = 600
DEFAULT_MAXIMUM_MEMORY_BYTES = 8 * 1024**3
DEFAULT_MAXIMUM_WORKING_STORAGE_BYTES = 5 * 1024**3
TYPOGRAPHIC_QUOTES = {"\u2018", "\u2019", "\u201c", "\u201d"}
RUNTIME_COMMANDS = {"Python": "python3", "R": "Rscript"}
MINIMAL_PATH = "/usr/bin:/bin:/usr/sbin:/sbin"
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
RESERVED_HOST_PATHS = {
    "network-deny.sb",
    "stdout.log",
    "stderr.log",
    "attempt-record.json",
}


def canonical_json(value: Any) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")


def json_object_without_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    value: dict[str, Any] = {}
    for key, item in pairs:
        if key in value:
            raise ValueError(f"duplicate JSON key: {key}")
        value[key] = item
    return value


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def sha256_fd(file_descriptor: int) -> str:
    digest = hashlib.sha256()
    os.lseek(file_descriptor, 0, os.SEEK_SET)
    while chunk := os.read(file_descriptor, 1024 * 1024):
        digest.update(chunk)
    return digest.hexdigest()


def directory_descriptor(path: Path) -> int:
    return os.open(
        path,
        os.O_RDONLY
        | getattr(os, "O_DIRECTORY", 0)
        | getattr(os, "O_NOFOLLOW", 0)
        | getattr(os, "O_CLOEXEC", 0),
    )


def nofollow_identity(directory_fd: int, name: str) -> dict[str, Any]:
    try:
        details = os.stat(name, dir_fd=directory_fd, follow_symlinks=False)
    except FileNotFoundError:
        return {"kind": "missing"}
    if stat.S_ISREG(details.st_mode):
        kind = "file"
    elif stat.S_ISDIR(details.st_mode):
        kind = "directory"
    elif stat.S_ISLNK(details.st_mode):
        kind = "symlink"
    else:
        kind = "other"
    return {
        "kind": kind,
        "device": details.st_dev,
        "inode": details.st_ino,
        "mode": stat.S_IMODE(details.st_mode),
        "bytes": details.st_size,
    }


def same_open_file(directory_fd: int, name: str, file_descriptor: int) -> bool:
    path_details = nofollow_identity(directory_fd, name)
    opened = os.fstat(file_descriptor)
    return (
        path_details.get("kind") == "file"
        and path_details.get("device") == opened.st_dev
        and path_details.get("inode") == opened.st_ino
        and stat.S_ISREG(opened.st_mode)
    )


def open_new_reserved_file(directory_fd: int, name: str) -> int:
    if name not in RESERVED_HOST_PATHS:
        raise ValueError(f"unregistered host-reserved path: {name}")
    if nofollow_identity(directory_fd, name)["kind"] != "missing":
        raise ValueError(f"host-reserved path already exists: {name}")
    return os.open(
        name,
        os.O_WRONLY
        | os.O_CREAT
        | os.O_EXCL
        | getattr(os, "O_NOFOLLOW", 0)
        | getattr(os, "O_CLOEXEC", 0),
        0o600,
        dir_fd=directory_fd,
    )


def atomic_write_reserved(
    directory_fd: int,
    name: str,
    contents: bytes,
    *,
    allow_existing_regular: bool,
) -> dict[str, Any]:
    """Replace one reserved entry without following an attacker-controlled link."""

    if name not in RESERVED_HOST_PATHS:
        raise ValueError(f"unregistered host-reserved path: {name}")
    before = nofollow_identity(directory_fd, name)
    if before["kind"] != "missing" and not (
        allow_existing_regular and before["kind"] == "file"
    ):
        raise ValueError(f"host-reserved path has unsafe type: {name} ({before['kind']})")

    temporary_name = f".{name}.host-{secrets.token_hex(12)}"
    temporary_fd = -1
    try:
        temporary_fd = os.open(
            temporary_name,
            os.O_WRONLY
            | os.O_CREAT
            | os.O_EXCL
            | getattr(os, "O_NOFOLLOW", 0)
            | getattr(os, "O_CLOEXEC", 0),
            0o600,
            dir_fd=directory_fd,
        )
        view = memoryview(contents)
        while view:
            written = os.write(temporary_fd, view)
            view = view[written:]
        os.fsync(temporary_fd)
        if not stat.S_ISREG(os.fstat(temporary_fd).st_mode):
            raise ValueError(f"temporary reserved output is not regular: {name}")
        # renameat replaces the directory entry itself and never follows a
        # target symlink, including one substituted after the lstat above.
        os.replace(
            temporary_name,
            name,
            src_dir_fd=directory_fd,
            dst_dir_fd=directory_fd,
        )
        after = nofollow_identity(directory_fd, name)
        if after.get("kind") != "file":
            raise ValueError(f"host-reserved path was replaced during write: {name}")
        return after
    finally:
        if temporary_fd >= 0:
            os.close(temporary_fd)
        try:
            os.unlink(temporary_name, dir_fd=directory_fd)
        except FileNotFoundError:
            pass


def quarantine_reserved_entry(directory_fd: int, name: str) -> str | None:
    """Move one unsafe reserved entry aside without following its object type."""

    if name not in RESERVED_HOST_PATHS:
        raise ValueError(f"unregistered host-reserved path: {name}")
    before = nofollow_identity(directory_fd, name)
    if before["kind"] == "missing":
        return None
    quarantine_name = f".claimbounty-quarantine-{name}-{secrets.token_hex(12)}"
    os.replace(
        name,
        quarantine_name,
        src_dir_fd=directory_fd,
        dst_dir_fd=directory_fd,
    )
    after = nofollow_identity(directory_fd, quarantine_name)
    if (
        nofollow_identity(directory_fd, name)["kind"] != "missing"
        or after.get("device") != before.get("device")
        or after.get("inode") != before.get("inode")
    ):
        raise ValueError(f"could not quarantine unsafe reserved path: {name}")
    return quarantine_name


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


def effective_timeout_seconds(execution_policy: dict[str, Any]) -> int:
    return int(execution_policy["limits"]["timeoutSeconds"])


def effective_limits(execution_policy: dict[str, Any]) -> dict[str, int]:
    supplied = execution_policy["limits"]
    return {
        "maximumMemoryBytes": int(supplied["maximumMemoryBytes"]),
        "maximumWorkingStorageBytes": int(supplied["maximumWorkingStorageBytes"]),
    }


def trusted_runtime_candidates(kind: str) -> list[Path]:
    if kind == "Python":
        return [Path(sys.executable).resolve()]
    if kind == "R":
        candidates = [
            Path("/usr/bin/Rscript"),
            Path("/usr/local/bin/Rscript"),
            Path("/opt/homebrew/bin/Rscript"),
            Path("/Library/Frameworks/R.framework/Resources/bin/Rscript"),
        ]
        return sorted(
            {candidate.resolve() for candidate in candidates if candidate.is_file()},
            key=lambda candidate: str(candidate),
        )
    return []


def resolve_allowlisted_runtime(execution_policy: dict[str, Any]) -> dict[str, Any]:
    runtime = execution_policy["runtime"]
    expected_path = Path(runtime["executable"]).resolve()
    candidates = trusted_runtime_candidates(runtime["kind"])
    if expected_path not in candidates:
        raise ValueError(
            "executionPolicy.runtime.executable is not an allowlisted runner-resolved runtime"
        )
    if not expected_path.is_file():
        raise ValueError("executionPolicy.runtime.executable is not a regular file")
    actual_sha256 = sha256_file(expected_path)
    if actual_sha256 != runtime["executableSha256"]:
        raise ValueError("executionPolicy runtime SHA-256 does not match the pinned runtime")
    return {
        **runtime,
        "executable": str(expected_path),
        "actualExecutableSha256": actual_sha256,
    }


def minimal_environment(
    attempt_directory: Path,
    execution_policy: dict[str, Any],
) -> dict[str, str]:
    runtime = execution_policy["runtime"]
    environment = {
        "HOME": str(attempt_directory),
        "LANG": "C.UTF-8",
        "LC_ALL": "C.UTF-8",
        "PATH": MINIMAL_PATH,
        "TMPDIR": str(attempt_directory),
    }
    library = runtime.get("libraryPath")
    if isinstance(library, dict):
        environment["R_LIBS_USER"] = str(Path(library["path"]).resolve())
    if runtime["kind"] == "R":
        environment["R_MAX_VSIZE"] = str(
            execution_policy["limits"]["maximumMemoryBytes"]
        )
    return environment


def frozen_execution_policy(manifest: dict[str, Any]) -> tuple[dict[str, Any], dict[str, Any]]:
    reference = manifest.get("executionPolicy")
    if not isinstance(reference, dict) or set(reference) != {"path", "sha256"}:
        raise ValueError("executionPolicy must contain only path and sha256")
    policy_path_value = reference.get("path")
    expected_sha256 = reference.get("sha256")
    if not isinstance(policy_path_value, str) or not policy_path_value.strip():
        raise ValueError("executionPolicy.path must be a non-empty string")
    if not isinstance(expected_sha256, str) or not re.fullmatch(r"[0-9a-f]{64}", expected_sha256):
        raise ValueError("executionPolicy.sha256 must be a lowercase SHA-256 digest")

    policy_reference_path = Path(policy_path_value)
    if policy_reference_path.is_symlink():
        raise ValueError("executionPolicy.path must not be a symlink")
    policy_path = policy_reference_path.resolve()
    if not policy_path.is_file():
        raise ValueError("executionPolicy.path must identify a regular, non-symlink file")
    try:
        policy_contents = policy_path.read_bytes()
    except OSError as exc:
        raise ValueError(f"cannot read executionPolicy: {exc}") from exc
    actual_sha256 = hashlib.sha256(policy_contents).hexdigest()
    if actual_sha256 != expected_sha256:
        raise ValueError("executionPolicy SHA-256 does not match the frozen policy")
    try:
        policy = json.loads(
            policy_contents,
            object_pairs_hook=json_object_without_duplicates,
        )
    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
        raise ValueError(f"cannot parse executionPolicy: {exc}") from exc
    if not isinstance(policy, dict):
        raise ValueError("executionPolicy must be a JSON object")
    if set(policy) != {
        "schemaVersion",
        "policyVersion",
        "runClass",
        "releaseScope",
        "outputRoot",
        "runtime",
        "sandbox",
        "limits",
        "mounts",
    }:
        raise ValueError("executionPolicy must contain the complete supported profile only")
    if policy.get("schemaVersion") != EXECUTION_POLICY_SCHEMA_VERSION:
        raise ValueError(
            f"executionPolicy.schemaVersion must equal {EXECUTION_POLICY_SCHEMA_VERSION}"
        )
    if not isinstance(policy.get("policyVersion"), str) or not policy["policyVersion"]:
        raise ValueError("executionPolicy.policyVersion must be a non-empty string")
    if policy.get("runClass") != "manual_local_operator":
        raise ValueError("executionPolicy.runClass must equal manual_local_operator")
    if policy.get("releaseScope") != "internal":
        raise ValueError("executionPolicy.releaseScope must equal internal")
    output_root_value = policy.get("outputRoot")
    if not isinstance(output_root_value, str) or not output_root_value.strip():
        raise ValueError("executionPolicy.outputRoot must be a non-empty path")
    output_root_reference = Path(output_root_value)
    if not output_root_reference.is_absolute() or output_root_reference.is_symlink():
        raise ValueError("executionPolicy.outputRoot must be a non-symlink directory")
    output_root = output_root_reference.resolve()
    if not output_root.is_dir():
        raise ValueError("executionPolicy.outputRoot must identify an existing directory")
    if output_root in {Path("/"), Path.home().resolve()}:
        raise ValueError("executionPolicy.outputRoot is broader than an admissible project root")
    sandbox = policy.get("sandbox")
    if not isinstance(sandbox, dict) or set(sandbox) != {
        "isolationRequired",
        "networkDuringAnalysis",
        "dependencyAcquisition",
        "expandArchivesAutomatically",
    }:
        raise ValueError("executionPolicy.sandbox must contain the complete supported controls")
    if sandbox.get("isolationRequired") is not True:
        raise ValueError("executionPolicy must require sandbox isolation")
    if sandbox.get("networkDuringAnalysis") != "disabled":
        raise ValueError("executionPolicy must disable network during analysis")
    if sandbox.get("dependencyAcquisition") != "disabled":
        raise ValueError("executionPolicy must disable dependency acquisition")
    if sandbox.get("expandArchivesAutomatically") is not False:
        raise ValueError("executionPolicy must disable automatic archive expansion")

    runtime = policy.get("runtime")
    required_runtime_keys = {
        "kind",
        "command",
        "executable",
        "executableSha256",
        "version",
        "requiredPackages",
        "readRoots",
    }
    if not isinstance(runtime, dict) or not required_runtime_keys.issubset(runtime):
        raise ValueError("executionPolicy.runtime must define the pinned runtime profile")
    allowed_runtime_keys = required_runtime_keys | {"libraryPath"}
    if set(runtime) - allowed_runtime_keys:
        raise ValueError("executionPolicy.runtime contains unsupported fields")
    kind = runtime.get("kind")
    if kind not in RUNTIME_COMMANDS or runtime.get("command") != RUNTIME_COMMANDS[kind]:
        raise ValueError("executionPolicy.runtime command is not allowlisted for its kind")
    if not isinstance(runtime.get("executable"), str) or not runtime["executable"]:
        raise ValueError("executionPolicy.runtime.executable must be a non-empty path")
    if not isinstance(runtime.get("executableSha256"), str) or not re.fullmatch(
        r"[0-9a-f]{64}", runtime["executableSha256"]
    ):
        raise ValueError("executionPolicy.runtime.executableSha256 must be a lowercase SHA-256")
    if not isinstance(runtime.get("version"), str) or not re.fullmatch(
        r"\d+(?:\.\d+)+", runtime["version"]
    ):
        raise ValueError("executionPolicy.runtime.version must be an exact numeric version")
    packages = runtime.get("requiredPackages")
    if not isinstance(packages, list) or not all(
        isinstance(package, str) and package for package in packages
    ) or len(packages) != len(set(packages)):
        raise ValueError("executionPolicy.runtime.requiredPackages must be unique strings")
    read_roots = runtime.get("readRoots")
    if not isinstance(read_roots, list):
        raise ValueError("executionPolicy.runtime.readRoots must be an array")
    seen_runtime_roots: set[str] = set()
    broad_runtime_roots = {
        Path("/"),
        Path("/Applications"),
        Path("/Library"),
        Path("/System"),
        Path("/Users"),
        Path("/opt"),
        Path("/usr"),
        Path("/var"),
        Path.home().resolve(),
    }
    for read_root in read_roots:
        if not isinstance(read_root, dict) or set(read_root) != {"path", "kind", "sha256"}:
            raise ValueError("executionPolicy runtime read roots require path, kind, and sha256")
        read_path_value = read_root.get("path")
        if not isinstance(read_path_value, str) or not read_path_value.strip():
            raise ValueError("executionPolicy runtime read-root paths must be non-empty")
        read_reference = Path(read_path_value)
        if not read_reference.is_absolute() or read_reference.is_symlink():
            raise ValueError("executionPolicy runtime read roots must not be symlinks")
        resolved_read_root = read_reference.resolve()
        if resolved_read_root in broad_runtime_roots:
            raise ValueError("executionPolicy runtime read root is broader than a runtime closure")
        if str(resolved_read_root) in seen_runtime_roots:
            raise ValueError("executionPolicy runtime read roots must be unique")
        seen_runtime_roots.add(str(resolved_read_root))
        identity = path_identity(resolved_read_root)
        if identity["kind"] != read_root.get("kind"):
            raise ValueError("executionPolicy runtime read-root kind does not match")
        if identity["sha256"] != read_root.get("sha256"):
            raise ValueError("executionPolicy runtime read-root SHA-256 does not match")
    library = runtime.get("libraryPath")
    if library is not None:
        if not isinstance(library, dict) or set(library) != {"path", "sha256"}:
            raise ValueError("executionPolicy.runtime.libraryPath must contain path and sha256")
        if not isinstance(library.get("path"), str) or not library["path"]:
            raise ValueError("executionPolicy.runtime.libraryPath.path must be non-empty")
        if not isinstance(library.get("sha256"), str) or not re.fullmatch(
            r"[0-9a-f]{64}", library["sha256"]
        ):
            raise ValueError("executionPolicy.runtime.libraryPath.sha256 must be lowercase SHA-256")
        library_reference = Path(library["path"])
        if library_reference.is_symlink():
            raise ValueError("executionPolicy.runtime.libraryPath.path must not be a symlink")
        library_identity = path_identity(library_reference)
        if library_identity["kind"] != "directory":
            raise ValueError("executionPolicy.runtime.libraryPath.path must identify a directory")
        if library_identity["sha256"] != library["sha256"]:
            raise ValueError("executionPolicy runtime library SHA-256 does not match the profile")
        if str(library_reference.resolve()) not in seen_runtime_roots:
            raise ValueError("executionPolicy runtime library must be an explicit read root")

    limits = policy.get("limits")
    if not isinstance(limits, dict) or set(limits) != {
        "timeoutSeconds",
        "maximumMemoryBytes",
        "maximumWorkingStorageBytes",
    }:
        raise ValueError("executionPolicy.limits must fully define timeout, memory, and storage")
    limit_caps = {
        "timeoutSeconds": DEFAULT_TIMEOUT_SECONDS,
        "maximumMemoryBytes": DEFAULT_MAXIMUM_MEMORY_BYTES,
        "maximumWorkingStorageBytes": DEFAULT_MAXIMUM_WORKING_STORAGE_BYTES,
    }
    for key, cap in limit_caps.items():
        value = limits.get(key)
        if not isinstance(value, int) or isinstance(value, bool) or value <= 0:
            raise ValueError(f"executionPolicy.limits.{key} must be a positive integer")
        if value > cap:
            raise ValueError(f"executionPolicy.limits.{key} exceeds the runner safety cap")

    mounts = policy.get("mounts")
    if not isinstance(mounts, list):
        raise ValueError("executionPolicy.mounts must be an array")
    seen_mounts: set[str] = set()
    broad_roots = {
        Path("/"),
        Path("/bin"),
        Path("/dev"),
        Path("/etc"),
        Path("/Library"),
        Path("/opt"),
        Path("/sbin"),
        Path("/System"),
        Path("/usr"),
        Path("/Users"),
        Path("/Volumes"),
        Path("/home"),
        Path("/var"),
        Path("/private"),
        Path("/Applications"),
        Path.home().resolve(),
    }
    for mount in mounts:
        if not isinstance(mount, dict) or set(mount) != {
            "path",
            "mode",
            "verifyUnchanged",
            "sha256",
        }:
            raise ValueError(
                "executionPolicy mounts must contain path, mode, verifyUnchanged, and sha256"
            )
        path_value = mount.get("path")
        if not isinstance(path_value, str) or not path_value:
            raise ValueError("executionPolicy mount path must be non-empty")
        mount_reference = Path(path_value)
        if mount_reference.is_symlink():
            raise ValueError("executionPolicy mount paths must not be symlinks")
        mounted = mount_reference.resolve()
        if str(mounted) in seen_mounts:
            raise ValueError("executionPolicy mount paths must be unique")
        seen_mounts.add(str(mounted))
        if mounted in broad_roots:
            raise ValueError("executionPolicy mount path is broader than an admissible input")
        if mount.get("mode") != "read-only":
            raise ValueError(
                "executionPolicy mounts must be read-only; attemptDirectory is the only write boundary"
            )
        if mount.get("verifyUnchanged") is not True:
            raise ValueError("executionPolicy mounts must require unchanged-content verification")
        if not isinstance(mount.get("sha256"), str) or not re.fullmatch(
            r"[0-9a-f]{64}", mount["sha256"]
        ):
            raise ValueError("executionPolicy mount sha256 must be a lowercase SHA-256")

    return policy, {
        "path": str(policy_path),
        "expectedSha256": expected_sha256,
        "actualSha256": actual_sha256,
        "networkDuringAnalysis": "disabled",
        "profileSha256": actual_sha256,
    }


def trusted_attempt_directory(
    manifest: dict[str, Any], execution_policy: dict[str, Any]
) -> Path:
    output_root = Path(execution_policy["outputRoot"]).resolve()
    attempt_reference = Path(required_string(manifest, "attemptDirectory"))
    if not attempt_reference.is_absolute():
        raise ValueError("attemptDirectory must be absolute")
    if attempt_reference.is_symlink():
        raise ValueError("attemptDirectory must not be a symlink")
    absolute_attempt = Path(os.path.abspath(attempt_reference))
    attempt = absolute_attempt.parent.resolve() / absolute_attempt.name
    if attempt.parent != output_root or attempt.name in {"", ".", ".."}:
        raise ValueError("attemptDirectory must be one direct child of executionPolicy.outputRoot")
    return attempt


def ensure_attempt_directory(attempt_directory: Path) -> None:
    try:
        attempt_directory.mkdir(mode=0o700, parents=False, exist_ok=True)
    except OSError as exc:
        raise ValueError(f"cannot create trusted attempt directory: {exc}") from exc
    if attempt_directory.is_symlink() or not attempt_directory.is_dir():
        raise ValueError("trusted attempt directory is not a regular directory boundary")


def verified_input_identities(execution_policy: dict[str, Any]) -> list[dict[str, Any]]:
    return [
        path_identity(Path(mount["path"]))
        for mount in execution_policy["mounts"]
        if mount.get("verifyUnchanged") is True
    ]


def scan_output_directory(
    directory_fd: int,
    relative_parent: Path = Path(),
) -> tuple[list[dict[str, Any]], list[dict[str, str]]]:
    identities: list[dict[str, Any]] = []
    unsafe: list[dict[str, str]] = []
    try:
        entries = sorted(os.scandir(directory_fd), key=lambda item: item.name)
    except OSError as exc:
        return [], [{"path": relative_parent.as_posix(), "kind": f"scan-error:{exc.errno}"}]
    for entry in entries:
        relative = relative_parent / entry.name
        if relative == Path("attempt-record.json"):
            continue
        try:
            details = entry.stat(follow_symlinks=False)
            if stat.S_ISLNK(details.st_mode):
                unsafe.append({"path": relative.as_posix(), "kind": "symlink"})
                continue
            if stat.S_ISDIR(details.st_mode):
                child_fd = os.open(
                    entry.name,
                    os.O_RDONLY
                    | getattr(os, "O_DIRECTORY", 0)
                    | getattr(os, "O_NOFOLLOW", 0)
                    | getattr(os, "O_CLOEXEC", 0),
                    dir_fd=directory_fd,
                )
                try:
                    child_identities, child_unsafe = scan_output_directory(
                        child_fd, relative
                    )
                    identities.extend(child_identities)
                    unsafe.extend(child_unsafe)
                finally:
                    os.close(child_fd)
                continue
            if not stat.S_ISREG(details.st_mode):
                unsafe.append({"path": relative.as_posix(), "kind": "non-regular"})
                continue
            file_fd = os.open(
                entry.name,
                os.O_RDONLY
                | getattr(os, "O_NOFOLLOW", 0)
                | getattr(os, "O_CLOEXEC", 0),
                dir_fd=directory_fd,
            )
            try:
                opened = os.fstat(file_fd)
                if (
                    not stat.S_ISREG(opened.st_mode)
                    or opened.st_dev != details.st_dev
                    or opened.st_ino != details.st_ino
                ):
                    raise OSError("output object changed during no-follow scan")
                identities.append(
                    {
                        "path": relative.as_posix(),
                        "sha256": sha256_fd(file_fd),
                        "bytes": opened.st_size,
                    }
                )
            finally:
                os.close(file_fd)
        except OSError:
            unsafe.append({"path": relative.as_posix(), "kind": "changed-during-scan"})
    return identities, unsafe


def output_scan(attempt_directory: Path) -> tuple[list[dict[str, Any]], list[dict[str, str]]]:
    directory_fd = directory_descriptor(attempt_directory)
    try:
        return scan_output_directory(directory_fd)
    finally:
        os.close(directory_fd)


def output_identities(attempt_directory: Path) -> list[dict[str, Any]]:
    identities, _ = output_scan(attempt_directory)
    return identities


def command_r_source(
    manifest: dict[str, Any], working_directory: Path | None = None
) -> Path | None:
    for argument in manifest.get("command", [])[1:]:
        if isinstance(argument, str) and argument.lower().endswith(".r"):
            source = Path(argument)
            if not source.is_absolute() and working_directory is not None:
                source = working_directory / source
            return Path(os.path.abspath(source))
    return None


def unsafe_r_source_contents_failure(content: str) -> dict[str, str] | None:
    if any(character in content for character in TYPOGRAPHIC_QUOTES):
        return {
            "class": "invalid_wrapper",
            "message": "R command source contains a typographic quote; use an ASCII string literal encoder",
        }
    return None


def unsafe_r_source_failure(source: Path) -> dict[str, str] | None:
    try:
        content = source.read_text(encoding="utf-8")
    except OSError as exc:
        return {"class": "invalid_wrapper", "message": f"cannot read R command source: {exc}"}
    return unsafe_r_source_contents_failure(content)


def open_regular_file_below(directory: Path, relative: Path) -> int:
    parts = relative.parts
    if not parts or any(part in {"", ".", ".."} for part in parts):
        raise ValueError("invalid no-follow relative path")
    current_fd = directory_descriptor(directory)
    try:
        for part in parts[:-1]:
            next_fd = os.open(
                part,
                os.O_RDONLY
                | getattr(os, "O_DIRECTORY", 0)
                | getattr(os, "O_NOFOLLOW", 0)
                | getattr(os, "O_CLOEXEC", 0),
                dir_fd=current_fd,
            )
            os.close(current_fd)
            current_fd = next_fd
        file_fd = os.open(
            parts[-1],
            os.O_RDONLY
            | getattr(os, "O_NOFOLLOW", 0)
            | getattr(os, "O_CLOEXEC", 0),
            dir_fd=current_fd,
        )
        if not stat.S_ISREG(os.fstat(file_fd).st_mode):
            os.close(file_fd)
            raise ValueError("source is not a regular file")
        return file_fd
    finally:
        os.close(current_fd)


def open_allowed_r_source(
    source: Path,
    attempt_directory: Path,
    execution_policy: dict[str, Any],
) -> int:
    source_absolute = Path(os.path.abspath(source))
    allowed_directories = [attempt_directory.resolve()]
    allowed_files: list[Path] = []
    for mount in execution_policy["mounts"]:
        mounted = Path(mount["path"]).resolve()
        details = os.stat(mounted, follow_symlinks=False)
        if stat.S_ISDIR(details.st_mode):
            allowed_directories.append(mounted)
        elif stat.S_ISREG(details.st_mode):
            allowed_files.append(mounted)

    for allowed_file in allowed_files:
        if source_absolute == allowed_file:
            file_fd = os.open(
                allowed_file,
                os.O_RDONLY
                | getattr(os, "O_NOFOLLOW", 0)
                | getattr(os, "O_CLOEXEC", 0),
            )
            if not stat.S_ISREG(os.fstat(file_fd).st_mode):
                os.close(file_fd)
                break
            return file_fd

    for allowed_directory in sorted(
        set(allowed_directories), key=lambda path: len(path.parts), reverse=True
    ):
        try:
            relative = source_absolute.relative_to(allowed_directory)
        except ValueError:
            continue
        try:
            return open_regular_file_below(allowed_directory, relative)
        except (OSError, ValueError) as exc:
            raise ValueError(
                "R command source must be a regular no-follow file in an allowed execution root"
            ) from exc
    raise ValueError(
        "R command source must be a regular no-follow file in an allowed execution root"
    )


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
            "filesystemIsolation": (
                "sandbox-exec-read-write-boundary" if sandbox else "unavailable"
            ),
            "processIsolation": (
                "sandbox-exec-no-process-fork" if sandbox else "unavailable"
            ),
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
        value = json.loads(
            path.read_text(encoding="utf-8"),
            object_pairs_hook=json_object_without_duplicates,
        )
    except (OSError, json.JSONDecodeError, ValueError) as exc:
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
        supported_fields = {
            "protocolVersion",
            "runId",
            "attemptId",
            "attemptDirectory",
            "workingDirectory",
            "command",
            "executionPolicy",
            "environment",
            "limits",
            "mounts",
            "network",
            "runtime",
            "timeoutSeconds",
        }
        unsupported_fields = sorted(set(manifest) - supported_fields)
        if unsupported_fields:
            raise ValueError(
                "manifest contains unsupported fields: " + ", ".join(unsupported_fields)
            )
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
        forbidden_controls = {
            "environment",
            "limits",
            "mounts",
            "network",
            "runtime",
            "timeoutSeconds",
        }
        supplied_controls = sorted(forbidden_controls.intersection(manifest))
        if supplied_controls:
            raise ValueError(
                "execution controls are derived from executionPolicy and must not be supplied "
                f"independently: {', '.join(supplied_controls)}"
            )
        execution_policy = manifest.get("executionPolicy")
        if not isinstance(execution_policy, dict):
            raise ValueError("executionPolicy must be an object")
        attempt_reference = Path(manifest["attemptDirectory"])
        if attempt_reference.is_symlink():
            raise ValueError("attemptDirectory must not be a symlink")
        attempt = attempt_reference.resolve()
        working = Path(manifest["workingDirectory"]).resolve()
        if working != attempt and attempt not in working.parents:
            raise ValueError("workingDirectory must be inside attemptDirectory")
    except ValueError as exc:
        failures.append({"class": "invalid_manifest", "message": str(exc)})
    return failures


def run_isolated_probe(
    command: list[str],
    working_directory: Path,
    attempt_directory: Path,
    execution_policy: dict[str, Any],
    timeout: int = 60,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        sandbox_command(command, attempt_directory, execution_policy),
        cwd=working_directory,
        env=minimal_environment(attempt_directory, execution_policy),
        capture_output=True,
        text=True,
        check=False,
        timeout=timeout,
    )


def runtime_version(
    resolved_runtime: dict[str, Any],
    working_directory: Path,
    attempt_directory: Path,
    execution_policy: dict[str, Any],
) -> str:
    completed = run_isolated_probe(
        [resolved_runtime["executable"], "--version"],
        working_directory,
        attempt_directory,
        execution_policy,
    )
    combined = (completed.stdout + completed.stderr).strip()
    return combined.splitlines()[0] if combined else "version unavailable"


def version_tuple(value: str) -> tuple[int, ...]:
    match = re.search(r"\d+(?:\.\d+)+", value)
    return tuple(int(part) for part in match.group(0).split(".")) if match else ()


def package_probe(
    resolved_runtime: dict[str, Any],
    working_directory: Path,
    attempt_directory: Path,
    execution_policy: dict[str, Any],
) -> tuple[list[str], dict[str, str]]:
    required = resolved_runtime["requiredPackages"]
    versions: dict[str, str] = {}
    missing: list[str] = []
    if resolved_runtime["kind"] == "Python":
        for package in required:
            expression = (
                "import importlib.metadata as m, importlib.util as u, sys; "
                "p=sys.argv[1]; s=u.find_spec(p); "
                "print(m.version(p) if s is not None else ''); "
                "sys.exit(0 if s is not None else 42)"
            )
            completed = run_isolated_probe(
                [resolved_runtime["executable"], "-c", expression, package],
                working_directory,
                attempt_directory,
                execution_policy,
            )
            if completed.returncode != 0:
                missing.append(package)
            else:
                versions[package] = completed.stdout.strip().splitlines()[-1] or "present"
        return missing, versions

    for package in required:
        expression = (
            f"p <- {json.dumps(package)}; "
            "if (!requireNamespace(p, quietly=TRUE)) quit(status=42); "
            "cat(p, as.character(packageVersion(p)), sep='=', fill=TRUE)"
        )
        completed = run_isolated_probe(
            [resolved_runtime["executable"], "-e", expression],
            working_directory,
            attempt_directory,
            execution_policy,
        )
        if completed.returncode != 0:
            missing.append(package)
        else:
            line = completed.stdout.strip().splitlines()[-1]
            versions[package] = line.split("=", 1)[-1] if "=" in line else "present"
    return missing, versions


def parse_r_command_source(
    manifest: dict[str, Any],
    resolved_runtime: dict[str, Any],
    working_directory: Path,
    attempt_directory: Path,
    execution_policy: dict[str, Any],
) -> tuple[dict[str, Any] | None, dict[str, str] | None]:
    source = command_r_source(manifest, working_directory)
    if source is None:
        return None, None
    try:
        source_fd = open_allowed_r_source(source, attempt_directory, execution_policy)
    except (OSError, ValueError):
        failure = {
            "class": "invalid_wrapper",
            "message": "R command source must be a regular no-follow file in an allowed execution root",
        }
        return {"sourceAccepted": False, "succeeded": False}, failure
    try:
        source_sha256 = sha256_fd(source_fd)
        os.lseek(source_fd, 0, os.SEEK_SET)
        source_contents = b""
        while chunk := os.read(source_fd, 1024 * 1024):
            source_contents += chunk
        content = source_contents.decode("utf-8")
    except (OSError, UnicodeDecodeError):
        return {"sourceAccepted": True, "succeeded": False}, {
            "class": "invalid_wrapper",
            "message": "R command source could not be read as UTF-8",
        }
    finally:
        os.close(source_fd)
    unsafe = unsafe_r_source_contents_failure(content)
    if unsafe:
        return {"source": str(source), "sourceAccepted": True, "succeeded": False}, unsafe
    expression = f"parse(file={json.dumps(str(source))}, keep.source=TRUE); cat('runner-r-parse-ok\\n')"
    try:
        completed = run_isolated_probe(
            [resolved_runtime["executable"], "--vanilla", "-e", expression],
            working_directory,
            attempt_directory,
            execution_policy,
            timeout=min(effective_timeout_seconds(execution_policy), 60),
        )
    except subprocess.TimeoutExpired:
        failure = {
            "class": "invalid_wrapper",
            "message": "R command source parse check exceeded 60 seconds",
        }
        return {"source": str(source), "succeeded": False}, failure
    check = {
        "source": str(source),
        "sourceAccepted": True,
        "sourceSha256": source_sha256,
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

    try:
        execution_policy, policy_identity = frozen_execution_policy(manifest)
        checks["executionPolicy"] = policy_identity
        resolved_runtime = resolve_allowlisted_runtime(execution_policy)
    except ValueError as exc:
        failures.append({"class": "invalid_manifest", "message": str(exc)})
        return {"status": "blocked", "checks": checks, "failures": failures}

    try:
        attempt_directory = trusted_attempt_directory(manifest, execution_policy)
        ensure_attempt_directory(attempt_directory)
    except ValueError as exc:
        failures.append({"class": "invalid_path", "message": str(exc)})
        return {"status": "blocked", "checks": checks, "failures": failures}
    working_directory = Path(manifest["workingDirectory"]).resolve()
    if not working_directory.is_dir():
        failures.append({"class": "invalid_path", "message": "workingDirectory is not a directory"})
    elif not os.access(working_directory, os.R_OK | os.X_OK):
        failures.append({"class": "permission_denied", "message": "workingDirectory is not readable"})

    for mount in execution_policy["mounts"]:
        mount_path = Path(mount["path"]).resolve()
        if not mount_path.exists():
            failures.append({"class": "missing_file", "message": f"mount is missing: {mount_path}"})
        elif not os.access(mount_path, os.R_OK):
            failures.append({"class": "permission_denied", "message": f"mount is unreadable: {mount_path}"})
        elif mount["mode"] == "read-write" and not os.access(mount_path, os.W_OK):
            failures.append({"class": "permission_denied", "message": f"mount is not writable: {mount_path}"})

        mount_identity = path_identity(mount_path)
        if mount_identity["sha256"] != mount["sha256"]:
            failures.append(
                {
                    "class": "invalid_manifest",
                    "message": f"mount SHA-256 does not match frozen profile: {mount_path}",
                }
            )

    expected_command = execution_policy["runtime"]["command"]
    if manifest["command"][0] != expected_command:
        failures.append(
            {
                "class": "invalid_manifest",
                "message": f"command must start with profile runtime alias {expected_command}",
            }
        )

    if identity["controls"]["networkIsolation"] == "unavailable":
        failures.append({"class": "isolation_unavailable", "message": "sandbox-exec is required for disabled network"})
    if identity["controls"]["filesystemIsolation"] == "unavailable":
        failures.append(
            {
                "class": "isolation_unavailable",
                "message": "sandbox-exec is required to deny undeclared host reads and writes outside attemptDirectory",
            }
        )
    if identity["controls"]["processIsolation"] == "unavailable":
        failures.append(
            {
                "class": "isolation_unavailable",
                "message": "sandbox-exec is required to deny scientific child-process creation",
            }
        )

    try:
        with tempfile.NamedTemporaryFile(dir=attempt_directory, prefix="write-check-", delete=True):
            pass
        checks["attemptDirectoryWritable"] = True
    except OSError as exc:
        failures.append({"class": "permission_denied", "message": f"attempt directory is not writable: {exc}"})

    if not failures:
        try:
            version = runtime_version(
                resolved_runtime,
                working_directory,
                attempt_directory,
                execution_policy,
            )
        except (OSError, subprocess.TimeoutExpired) as exc:
            failures.append(
                {"class": "missing_dependency", "message": f"runtime version probe failed: {exc}"}
            )
            version = None
        checks["runtime"] = {
            "kind": resolved_runtime["kind"],
            "executable": resolved_runtime["executable"],
            "executableSha256": resolved_runtime["actualExecutableSha256"],
            "version": version,
            "pinnedVersion": resolved_runtime["version"],
        }
        if version is not None and version_tuple(version) != version_tuple(resolved_runtime["version"]):
            failures.append(
                {
                    "class": "version_conflict",
                    "message": (
                        f"runtime {version} does not equal pinned version "
                        f"{resolved_runtime['version']}"
                    ),
                }
            )

    if not failures:
        try:
            missing, versions = package_probe(
                resolved_runtime,
                working_directory,
                attempt_directory,
                execution_policy,
            )
        except (OSError, subprocess.TimeoutExpired) as exc:
            failures.append(
                {"class": "missing_dependency", "message": f"package probe failed: {exc}"}
            )
            missing, versions = [], {}
        checks["packages"] = versions
        if missing:
            failures.append(
                {
                    "class": "missing_dependency",
                    "message": "missing packages: " + ", ".join(missing),
                }
            )

    if not failures and resolved_runtime["kind"] == "R":
        parse_check, parse_failure = parse_r_command_source(
            manifest,
            resolved_runtime,
            working_directory,
            attempt_directory,
            execution_policy,
        )
        if parse_check is not None:
            checks["rCommandSourceParse"] = parse_check
        if parse_failure is not None:
            failures.append(parse_failure)

    if not failures:
        probe_command = (
            [resolved_runtime["executable"], "-e", "cat('runner-preflight-ok\\n')"]
            if resolved_runtime["kind"] == "R"
            else [resolved_runtime["executable"], "-c", "print('runner-preflight-ok')"]
        )
        completed = run_isolated_probe(
            probe_command,
            working_directory,
            attempt_directory,
            execution_policy,
        )
        checks["trivialCommand"] = {
            "command": probe_command,
            "exitCode": completed.returncode,
            "succeeded": completed.returncode == 0,
            "stderr": completed.stderr[-2000:],
        }
        if completed.returncode != 0:
            failures.append({"class": "runtime_error", "message": "trivial runtime command failed"})

    if identity["controls"]["networkIsolation"] != "unavailable" and not failures:
        network_probe = (
            [
                resolved_runtime["executable"],
                "-c",
                (
                    "import errno,socket,sys; denied=(errno.EPERM,errno.EACCES); "
                    "tcp=socket.socket(); tr=tcp.connect_ex(('127.0.0.1',9)); tcp.close(); "
                    "udp=socket.socket(socket.AF_INET,socket.SOCK_DGRAM); "
                    "ur=udp.connect_ex(('198.51.100.1',9)); udp.close(); "
                    "sys.exit(0 if tr in denied and ur in denied else 3)"
                ),
            ]
            if resolved_runtime["kind"] == "Python"
            else [
                resolved_runtime["executable"],
                "--vanilla",
                "-e",
                (
                    "x <- try(socketConnection('198.51.100.1', 9, open='w', timeout=1), "
                    "silent=TRUE); quit(status=if (inherits(x, 'try-error')) 0 else 3)"
                ),
            ]
        )
        completed = run_isolated_probe(
            network_probe,
            working_directory,
            attempt_directory,
            execution_policy,
        )
        checks["networkIsolationVerified"] = completed.returncode == 0
        if completed.returncode != 0:
            failures.append(
                {
                    "class": "isolation_unavailable",
                    "message": "network-denial probe did not return an enforced permission error",
                }
            )

    if identity["controls"]["filesystemIsolation"] != "unavailable" and not failures:
        sentinel_directory = attempt_directory.parent
        try:
            with tempfile.NamedTemporaryFile(
                dir=sentinel_directory,
                prefix="claimbounty-host-read-deny-",
                delete=True,
            ) as sentinel:
                sentinel.write(b"host-read-must-be-denied\n")
                sentinel.flush()
                host_file_probe = (
                    [
                        resolved_runtime["executable"],
                        "-c",
                        (
                            "import errno,sys; "
                            "\ntry:\n open(sys.argv[1], 'rb').read(); result=3"
                            "\nexcept OSError as exc:\n result=0 if exc.errno in (errno.EPERM,errno.EACCES) else 4"
                            "\nsys.exit(result)"
                        ),
                        sentinel.name,
                    ]
                    if resolved_runtime["kind"] == "Python"
                    else [
                        resolved_runtime["executable"],
                        "--vanilla",
                        "-e",
                        (
                            f"p <- {json.dumps(sentinel.name)}; "
                            "x <- try(readBin(p, 'raw', n=1), silent=TRUE); "
                            "quit(status=if (inherits(x, 'try-error')) 0 else 3)"
                        ),
                    ]
                )
                completed = run_isolated_probe(
                    host_file_probe,
                    working_directory,
                    attempt_directory,
                    execution_policy,
                )
        except OSError as exc:
            completed = None
            failures.append(
                {
                    "class": "isolation_unavailable",
                    "message": f"cannot create host-file denial probe: {exc}",
                }
            )
        checks["hostFileReadIsolationVerified"] = (
            completed is not None and completed.returncode == 0
        )
        if completed is not None and completed.returncode != 0:
            failures.append(
                {
                    "class": "isolation_unavailable",
                    "message": "host-file denial probe did not return an enforced permission error",
                }
            )

    if identity["controls"]["processIsolation"] != "unavailable" and not failures:
        process_probe = (
            [
                resolved_runtime["executable"],
                "-c",
                (
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
                ),
            ]
            if resolved_runtime["kind"] == "Python"
            else [
                resolved_runtime["executable"],
                "--vanilla",
                "-e",
                (
                    "x <- try(system2('/usr/bin/true'), silent=TRUE); "
                    "quit(status=if (inherits(x, 'try-error') || !identical(x, 0L)) 0 else 3)"
                ),
            ]
        )
        completed = run_isolated_probe(
            process_probe,
            working_directory,
            attempt_directory,
            execution_policy,
        )
        checks["processCreationIsolationVerified"] = completed.returncode == 0
        if completed.returncode != 0:
            failures.append(
                {
                    "class": "isolation_unavailable",
                    "message": (
                        "child-process denial probe could create a child or detached process"
                    ),
                }
            )

    checks["manifestSha256"] = hashlib.sha256(canonical_json(manifest)).hexdigest()
    checks["environmentSha256"] = hashlib.sha256(
        canonical_json(
            {
                "environment": minimal_environment(attempt_directory, execution_policy),
                "runtime": checks.get("runtime"),
                "packages": checks.get("packages", {}),
            }
        )
    ).hexdigest()
    checks["environmentKeys"] = sorted(minimal_environment(attempt_directory, execution_policy))
    checks["effectiveTimeoutSeconds"] = effective_timeout_seconds(execution_policy)
    checks["effectiveLimits"] = effective_limits(execution_policy)
    checks["inputIdentities"] = verified_input_identities(execution_policy)
    return {"status": "ready" if not failures else "blocked", "checks": checks, "failures": failures}


def directory_size(path: Path) -> int:
    identities, _ = output_scan(path)
    return sum(int(identity["bytes"]) for identity in identities)


def resident_bytes(pid: int) -> int | None:
    completed = subprocess.run(
        ["ps", "-o", "rss=", "-p", str(pid)], capture_output=True, text=True, check=False
    )
    try:
        return int(completed.stdout.strip()) * 1024
    except ValueError:
        return None


def sandbox_command(
    command: list[str],
    attempt_directory: Path,
    execution_policy: dict[str, Any],
) -> list[str]:
    if platform.system() != "Darwin" or not shutil.which("sandbox-exec"):
        return list(command)
    profile = attempt_directory / "network-deny.sb"
    if execution_policy["sandbox"]["networkDuringAnalysis"] != "disabled":
        raise ValueError("executionPolicy attempted to weaken network isolation")
    readable_directories = [attempt_directory]
    readable_files = [Path(execution_policy["runtime"]["executable"]).resolve()]
    for read_root in execution_policy["runtime"]["readRoots"]:
        target = Path(read_root["path"]).resolve()
        if read_root["kind"] == "directory":
            readable_directories.append(target)
        else:
            readable_files.append(target)
    for mount in execution_policy["mounts"]:
        target = Path(mount["path"]).resolve()
        if target.is_dir():
            readable_directories.append(target)
        else:
            readable_files.append(target)
    unique_readable_directories = sorted(
        {str(path.resolve()) for path in readable_directories}
    )
    unique_readable_files = sorted({str(path.resolve()) for path in readable_files})
    rules = [
        "(version 1)",
        "(deny default)",
        '(import "system.sb")',
        "(allow process-exec)",
        "(allow signal (target self))",
        "(allow file-read-metadata file-test-existence)",
        *[
            f"(allow file-read* file-map-executable (subpath {json.dumps(path)}))"
            for path in unique_readable_directories
        ],
        *[
            f"(allow file-read* file-map-executable (literal {json.dumps(path)}))"
            for path in unique_readable_files
        ],
        f"(allow file-write* (subpath {json.dumps(str(attempt_directory))}))",
        "(deny network*)",
    ]
    profile_contents = ("\n".join(rules) + "\n").encode("utf-8")
    attempt_fd = directory_descriptor(attempt_directory)
    try:
        atomic_write_reserved(
            attempt_fd,
            profile.name,
            profile_contents,
            allow_existing_regular=True,
        )
    finally:
        os.close(attempt_fd)
    return ["sandbox-exec", "-f", str(profile), *list(command)]


def execute_once(manifest: dict[str, Any]) -> dict[str, Any]:
    readiness = preflight(manifest)
    if readiness["status"] != "ready":
        return {"status": "blocked", "preflight": readiness, "failureClass": readiness["failures"][0]["class"]}

    execution_policy_for_boundary, _ = frozen_execution_policy(manifest)
    attempt_directory = trusted_attempt_directory(manifest, execution_policy_for_boundary)
    working_directory = Path(manifest["workingDirectory"]).resolve()
    stdout_path = attempt_directory / "stdout.log"
    stderr_path = attempt_directory / "stderr.log"
    try:
        execution_policy, policy_identity = frozen_execution_policy(manifest)
        resolved_runtime = resolve_allowlisted_runtime(execution_policy)
    except ValueError as exc:
        return {
            "status": "blocked",
            "failureClass": "invalid_manifest",
            "message": f"execution profile changed after preflight: {exc}",
        }
    if policy_identity != readiness["checks"]["executionPolicy"]:
        return {
            "status": "blocked",
            "failureClass": "invalid_manifest",
            "message": "executionPolicy changed after preflight",
        }
    limits = effective_limits(execution_policy)
    timeout_seconds = effective_timeout_seconds(execution_policy)
    maximum_memory = limits["maximumMemoryBytes"]
    maximum_storage = limits["maximumWorkingStorageBytes"]
    scientific_command = [resolved_runtime["executable"], *manifest["command"][1:]]
    attempt_fd = -1
    stdout_fd = -1
    stderr_fd = -1
    try:
        command = sandbox_command(scientific_command, attempt_directory, execution_policy)
        attempt_fd = directory_descriptor(attempt_directory)
        stdout_fd = open_new_reserved_file(attempt_fd, stdout_path.name)
        stderr_fd = open_new_reserved_file(attempt_fd, stderr_path.name)
        if nofollow_identity(attempt_fd, "attempt-record.json")["kind"] != "missing":
            raise ValueError("host-reserved path already exists: attempt-record.json")
        profile_identity = nofollow_identity(attempt_fd, "network-deny.sb")
        profile_sha256: str | None = None
        if profile_identity["kind"] != "missing":
            if profile_identity["kind"] != "file":
                raise ValueError("host-reserved path is not regular: network-deny.sb")
            profile_fd = os.open(
                "network-deny.sb",
                os.O_RDONLY
                | getattr(os, "O_NOFOLLOW", 0)
                | getattr(os, "O_CLOEXEC", 0),
                dir_fd=attempt_fd,
            )
            try:
                profile_sha256 = sha256_fd(profile_fd)
            finally:
                os.close(profile_fd)
    except (OSError, ValueError) as exc:
        for descriptor in (stdout_fd, stderr_fd, attempt_fd):
            if descriptor >= 0:
                os.close(descriptor)
        return {
            "status": "blocked",
            "failureClass": "isolation_unavailable",
            "message": f"cannot reserve host-written attempt paths: {exc}",
        }
    environment = minimal_environment(attempt_directory, execution_policy)

    def set_limits() -> None:
        os.setsid()
        try:
            resource.setrlimit(resource.RLIMIT_AS, (maximum_memory, maximum_memory))
        except (ValueError, OSError):
            pass

    started = time.time()
    monotonic_started = time.monotonic()
    termination: str | None = None
    with os.fdopen(os.dup(stdout_fd), "wb") as stdout, os.fdopen(
        os.dup(stderr_fd), "wb"
    ) as stderr:
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
    reserved_path_violations: list[str] = []
    if not same_open_file(attempt_fd, stdout_path.name, stdout_fd):
        reserved_path_violations.append(stdout_path.name)
    if not same_open_file(attempt_fd, stderr_path.name, stderr_fd):
        reserved_path_violations.append(stderr_path.name)
    if profile_sha256 is not None:
        try:
            profile_fd = os.open(
                "network-deny.sb",
                os.O_RDONLY
                | getattr(os, "O_NOFOLLOW", 0)
                | getattr(os, "O_CLOEXEC", 0),
                dir_fd=attempt_fd,
            )
            try:
                if sha256_fd(profile_fd) != profile_sha256:
                    reserved_path_violations.append("network-deny.sb")
            finally:
                os.close(profile_fd)
        except OSError:
            reserved_path_violations.append("network-deny.sb")
    reserved_path_quarantines: list[str] = []
    if nofollow_identity(attempt_fd, "attempt-record.json")["kind"] != "missing":
        reserved_path_violations.append("attempt-record.json")
        quarantined = quarantine_reserved_entry(attempt_fd, "attempt-record.json")
        if quarantined is not None:
            reserved_path_quarantines.append(quarantined)
    if reserved_path_violations:
        termination = "reserved_path_replaced"
    os.close(stdout_fd)
    os.close(stderr_fd)

    input_identities_before = readiness["checks"].get("inputIdentities", [])
    input_identities_after = verified_input_identities(execution_policy)
    try:
        _, execution_policy_after = frozen_execution_policy(manifest)
    except ValueError:
        execution_policy_after = None
    execution_policy_unchanged = (
        execution_policy_after == readiness["checks"]["executionPolicy"]
    )
    source_inputs_unchanged = (
        input_identities_before == input_identities_after and execution_policy_unchanged
    )
    if not source_inputs_unchanged:
        termination = "source_modified"
    output_identity_records, unsafe_output_objects = output_scan(attempt_directory)
    if unsafe_output_objects and termination is None:
        termination = "unsafe_output_object"
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
        "executionPolicy": readiness["checks"]["executionPolicy"],
        "executionPolicyAfter": execution_policy_after,
        "executionPolicyUnchanged": execution_policy_unchanged,
        "isolationChecks": {
            "networkDenied": readiness["checks"].get("networkIsolationVerified", False),
            "hostFileReadDenied": readiness["checks"].get(
                "hostFileReadIsolationVerified", False
            ),
            "childProcessCreationDenied": readiness["checks"].get(
                "processCreationIsolationVerified", False
            ),
        },
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
        "reservedPathViolations": sorted(set(reserved_path_violations)),
        "reservedPathQuarantines": reserved_path_quarantines,
        "unsafeOutputObjects": unsafe_output_objects,
        "stdout": str(stdout_path),
        "stderr": str(stderr_path),
    }
    record["outputIdentities"] = output_identity_records
    rendered_record = (json.dumps(record, indent=2, sort_keys=True) + "\n").encode("utf-8")
    try:
        atomic_write_reserved(
            attempt_fd,
            "attempt-record.json",
            rendered_record,
            allow_existing_regular=False,
        )
    finally:
        os.close(attempt_fd)
    return record


def fallback_terminal_write(
    attempt_directory: Path,
    result: dict[str, Any],
    *,
    replace_existing: bool = False,
) -> tuple[bool, str | None]:
    """Best-effort terminal persistence independent of the normal record writer."""

    attempt_fd = -1
    temporary_name = f".attempt-record.failure-{secrets.token_hex(12)}"
    temporary_fd = -1
    try:
        attempt_fd = directory_descriptor(attempt_directory)
        existing = nofollow_identity(attempt_fd, "attempt-record.json")
        if existing["kind"] == "file" and not replace_existing:
            return True, None
        if existing["kind"] != "missing":
            quarantined = quarantine_reserved_entry(attempt_fd, "attempt-record.json")
            if quarantined is not None:
                result.setdefault("reservedPathQuarantines", []).append(quarantined)
        rendered = (json.dumps(result, indent=2, sort_keys=True) + "\n").encode("utf-8")
        temporary_fd = os.open(
            temporary_name,
            os.O_WRONLY
            | os.O_CREAT
            | os.O_EXCL
            | getattr(os, "O_NOFOLLOW", 0)
            | getattr(os, "O_CLOEXEC", 0),
            0o600,
            dir_fd=attempt_fd,
        )
        view = memoryview(rendered)
        while view:
            written = os.write(temporary_fd, view)
            view = view[written:]
        os.fsync(temporary_fd)
        os.replace(
            temporary_name,
            "attempt-record.json",
            src_dir_fd=attempt_fd,
            dst_dir_fd=attempt_fd,
        )
        if nofollow_identity(attempt_fd, "attempt-record.json")["kind"] != "file":
            raise OSError("terminal attempt record did not become a regular file")
        return True, None
    except Exception as exc:
        return False, f"{type(exc).__name__}: {exc}"
    finally:
        if temporary_fd >= 0:
            os.close(temporary_fd)
        if attempt_fd >= 0:
            try:
                os.unlink(temporary_name, dir_fd=attempt_fd)
            except OSError:
                pass
            os.close(attempt_fd)


def execute(manifest: dict[str, Any]) -> dict[str, Any]:
    try:
        execution_policy, _ = frozen_execution_policy(manifest)
        attempt_directory = trusted_attempt_directory(manifest, execution_policy)
        ensure_attempt_directory(attempt_directory)
    except (OSError, ValueError) as exc:
        return {
            "status": "blocked",
            "failureClass": "invalid_manifest",
            "terminationReason": "attempt_boundary_unavailable",
            "message": str(exc),
            "recordPersisted": False,
        }

    try:
        result = execute_once(manifest)
    except Exception as exc:
        result = {
            "status": "failed",
            "failureClass": "runtime_error",
            "terminationReason": "operational_failure",
            "operationalFailure": {
                "type": type(exc).__name__,
                "message": str(exc),
            },
        }

    persisted, persistence_error = fallback_terminal_write(attempt_directory, result)
    result["recordPersisted"] = persisted
    if persistence_error is not None:
        result["recordPersistenceError"] = persistence_error
    return result


def trusted_cli_output(output: Path, trusted_root: Path) -> tuple[int, str]:
    root_reference = trusted_root
    if not root_reference.is_absolute() or root_reference.is_symlink():
        raise ValueError("trusted output root must be an absolute non-symlink directory")
    root = root_reference.resolve()
    if not root.is_dir() or root in {Path("/"), Path.home().resolve()}:
        raise ValueError("trusted output root must be an existing bounded directory")
    if not output.is_absolute():
        raise ValueError("output path must be absolute")
    output_absolute = Path(os.path.abspath(output))
    if output_absolute.parent != root or output_absolute.name in {"", ".", ".."}:
        raise ValueError("output path must be one direct child of the trusted output root")
    return directory_descriptor(root), output_absolute.name


def emit(value: dict[str, Any], output: Path | None, trusted_root: Path | None) -> None:
    rendered = json.dumps(value, indent=2, sort_keys=True) + "\n"
    if output:
        if trusted_root is None:
            raise ValueError("--output requires a trusted output root")
        root_fd, output_name = trusted_cli_output(output, trusted_root)
        temporary_name = f".{output_name}.host-{secrets.token_hex(12)}"
        temporary_fd = -1
        try:
            existing = nofollow_identity(root_fd, output_name)
            if existing["kind"] not in {"missing", "file"}:
                raise ValueError("output path has an unsupported filesystem type")
            temporary_fd = os.open(
                temporary_name,
                os.O_WRONLY
                | os.O_CREAT
                | os.O_EXCL
                | getattr(os, "O_NOFOLLOW", 0)
                | getattr(os, "O_CLOEXEC", 0),
                0o600,
                dir_fd=root_fd,
            )
            view = memoryview(rendered.encode("utf-8"))
            while view:
                written = os.write(temporary_fd, view)
                view = view[written:]
            os.fsync(temporary_fd)
            os.replace(
                temporary_name,
                output_name,
                src_dir_fd=root_fd,
                dst_dir_fd=root_fd,
            )
        finally:
            if temporary_fd >= 0:
                os.close(temporary_fd)
            try:
                os.unlink(temporary_name, dir_fd=root_fd)
            except FileNotFoundError:
                pass
            os.close(root_fd)
    sys.stdout.write(rendered)


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="operation", required=True)
    identify_parser = subparsers.add_parser("identify")
    identify_parser.add_argument("--output", type=Path)
    identify_parser.add_argument("--trusted-output-root", type=Path)
    for operation in ("preflight", "run"):
        operation_parser = subparsers.add_parser(operation)
        operation_parser.add_argument("--manifest", required=True, type=Path)
        operation_parser.add_argument("--output", type=Path)
        operation_parser.add_argument("--trusted-output-root", type=Path)
    args = parser.parse_args()

    if args.operation == "identify":
        result = runner_identity()
        trusted_output_root = args.trusted_output_root
    else:
        try:
            manifest = load_manifest(args.manifest)
            result = preflight(manifest) if args.operation == "preflight" else execute(manifest)
            if args.trusted_output_root is not None:
                trusted_output_root = args.trusted_output_root
            else:
                execution_policy, _ = frozen_execution_policy(manifest)
                trusted_output_root = Path(execution_policy["outputRoot"])
        except Exception as exc:
            result = {"status": "blocked", "failureClass": "invalid_manifest", "message": str(exc)}
            trusted_output_root = args.trusted_output_root
    try:
        emit(result, args.output, trusted_output_root)
    except (OSError, ValueError) as exc:
        result = {
            "status": "failed",
            "failureClass": "runtime_error",
            "terminationReason": "output_persistence_failure",
            "message": str(exc),
        }
        if args.operation == "run" and "manifest" in locals():
            try:
                execution_policy, _ = frozen_execution_policy(manifest)
                attempt_directory = trusted_attempt_directory(manifest, execution_policy)
                persisted, persistence_error = fallback_terminal_write(
                    attempt_directory,
                    result,
                    replace_existing=True,
                )
                result["recordPersisted"] = persisted
                if persistence_error is not None:
                    result["recordPersistenceError"] = persistence_error
            except Exception as persistence_exc:
                result["recordPersisted"] = False
                result["recordPersistenceError"] = str(persistence_exc)
        sys.stdout.write(json.dumps(result, indent=2, sort_keys=True) + "\n")
    return 0 if result.get("status") in {None, "ready", "completed"} else 2


if __name__ == "__main__":
    raise SystemExit(main())
