<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Non-runnable FCAF definitions

The FCAF catalog deliberately skips every directory named `_implementation`.
Definitions here preserve source-backed implementation work without exposing an
incomplete scenario as a runnable conformance test.

Definition states:

```text
tests/*.yaml                                  runnable and source-exact
tests/_implementation/pending/*.yaml          definition or evidence incomplete
tests/_implementation/verifier-blocked/*.yaml exact scenario needs verifier support
```

Files below `_implementation` retain their original FCAF ID and source path.
Moving a file directly into `tests/` is its activation event and requires every
activation gate in
`docs/superpowers/specs/2026-08-25-fcaf-test-remediation-design.md` to pass.

