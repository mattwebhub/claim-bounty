# Peer2Paper scientific execution runner

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
- The working directory must be inside the attempt directory. Copy or materialize
  the required source and data there before execution; never run from the source
  directory.
- The sandbox denies writes outside the attempt directory, so source mounts stay
  unchanged and working-storage accounting covers every permitted write.
- `network: host-policy` uses the host's normal network policy. Verified network
  and DNS denial is not a prerequisite for this workflow.
- `network: disabled` remains available when the host can enforce it, but it is
  not the default expected by the routine.
- Commands stop after `timeoutSeconds`, which defaults to 600 seconds.
- Memory and working-storage limits are operational defaults of 8 GiB and 5 GiB.
  They stop runaway processes but are not scientific-validity gates.
- R command files receive a parse-only check before execution. Generated wrappers
  containing typographic quotes are rejected before supplied scientific code runs.
- Mounts marked `verifyUnchanged` are hashed before and after execution. Results
  record input, command, output, runner, manifest, and environment identities.

Missing runtimes, packages, files, permissions, or an attempt-local working copy
remain concrete execution blockers. Network-denial availability and exact memory
or storage enforcement do not decide scientific assessability.

## Contract

`execution-manifest.schema.json` defines the accepted manifest. The current
protocol version is `1.0.0`. R and Python are supported. Other languages require
an explicit protocol revision.
