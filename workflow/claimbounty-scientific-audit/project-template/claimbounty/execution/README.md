# ClaimBounty scientific execution runner

> **Status**: Production | **Updated**: 2026-08-31 | **Scope**: Public ClaimBounty workflow export

`runner.py` is the shared local execution boundary for reproduction,
robustness, and independent statistical verification. Those stages must use the
same runner and the execution profile frozen during reproduction.

The runner accepts one versioned JSON manifest and supports three commands:

```text
python3 runner.py identify
python3 runner.py preflight --manifest MANIFEST.json
python3 runner.py run --manifest MANIFEST.json
```

It performs one execution attempt. It does not select repairs, retry failed
commands, manage workflow state, or decide scientific outcomes.

## Execution controls

- A dedicated attempt directory keeps outputs from different attempts separate.
- The frozen profile declares one existing `outputRoot`. Each attempt directory
  must be one direct child of that root, and the runner never creates arbitrary
  host parents. Optional `--output` files must be direct children of that root,
  or of an explicit absolute `--trusted-output-root` supplied by the operator.
- The working directory must be inside the attempt directory. Copy or materialize
  the required source and data there before execution; never run from the source
  directory.
- `executionPolicy.path` and `executionPolicy.sha256` bind every attempt to the
  complete frozen execution profile. The profile owns the pinned runtime,
  package list, mounts, timeout, memory and storage limits, and isolation
  controls. The manifest cannot override those controls.
- The runner resolves Python from its own interpreter and R from a fixed host
  path allowlist, then verifies the profile's executable digest and exact
  version. Every additional runtime file or directory needed for interpreter
  startup is listed in `runtime.readRoots` and hash-verified before use. Runtime
  version, package, parse-only, and trivial-command probes all run inside the
  same sandbox used for the scientific command.
- Analysis networking is derived from
  `executionPolicy.sandbox.networkDuringAnalysis`. The only accepted value is
  `disabled`; an independent manifest network setting or a weaker frozen policy
  blocks execution.
- On the supported macOS host, `sandbox-exec` denies network access, limits reads
  to macOS runtime support paths plus the attempt directory and frozen mounts,
  denies writes outside the attempt directory, and denies scientific
  child-process creation. Preflight verifies network denial, denial of an
  undeclared host-file read, and denial of both a fork and detached child with
  live probes before any study command runs. The supported workload is therefore
  single-process; studies that require subprocesses need a new frozen policy and
  stronger descendant supervision before they can execute.
- Commands stop at the frozen profile timeout, capped at 600 seconds. Frozen
  memory and working-storage limits cannot exceed 8 GiB and 5 GiB.
- Scientific processes receive a fixed minimal environment containing only the
  attempt-local home and temporary directory, locale, a system path, and
  runtime-specific settings derived from the profile. Ambient credentials and
  other host variables are not inherited.
- R command files receive a parse-only check before execution. Generated wrappers
  containing typographic quotes are rejected before supplied scientific code runs.
- Mounts marked `verifyUnchanged` are hashed before and after execution. Results
  record input, command, output, runner, manifest, and environment identities.

Missing runtimes, packages, files, permissions, or an attempt-local working copy
remain concrete execution blockers. Missing or failed network, host-file, or
child-process denial is also an execution blocker. Exact memory or storage enforcement does not
decide scientific assessability.

This runner records a verified macOS `sandbox-exec` boundary. It does not claim
that the study ran in a disposable virtual machine or container.

## Contract

`execution-manifest.schema.json` defines the accepted manifest and
`execution-profile.schema.json` defines the frozen profile. The current protocol
version is `1.2.0`, and the execution-profile schema is `1.2.0`. R and Python
are supported. Other languages require an explicit protocol revision.
