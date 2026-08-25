<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Non-runnable FCAF artifacts

This directory preserves pipeline, precondition, and batch artifacts belonging
to pending or verifier-blocked definitions.

- `pipelines/` is outside the top-level pipeline glob used by `fcaf-sync`.
- `preconditions/` is outside the active catalog precondition directory.
- `batches/` contains manifests that must not advertise runnable tests.

Artifacts move back to their active directory only with a source-exact test
definition and completed validation evidence.
