<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# FCAF parallel-work memory

Temporary coordination state for agents implementing FCAF definitions. Read `.agents/skills/fcaf-definitions/SKILL.md` for the durable workflow. Update this file as work advances; do not put credentials or secrets here.

Upstream and local quality findings are maintained as copy-paste-ready issue drafts in `pkg/fcaf/TEST-AUTHOR-FEEDBACK.md`.

## Git state at handoff

- Repository: `/home/puria/src/github.com/ForkbombEu/credimi/PR/1295`
- Detached HEAD: `54373c673c4d2e65118df2f77d32642dfba16e97`
- Shared push target: `origin HEAD:feat/fcaf-test`
- Worktree was clean before adding this memory and skill.
- Catalog count: 177 tests.

Recent commits:

- `54373c67` case 089, missing trusted-authority type, plus `request_rejected` validator mode
- `016077d6` case 088, unsupported trusted-authority type
- `b9978313` case 087, repeated queries matching the same PID
- `49822612` case 086
- `031de948` case 085
- `6c9d1fb9` case 084
- `5e80ac16` case 083
- `5821662d` case 082
- `12c466dc` case 081
- `d5c029ac` case 080

## Current scope

Implement mandatory wallet-solution/relying-party tests one at a time. Skip TSL/MTSL, Digital Credentials API, W3C Digital Credentials API, and CAW tests. Keep reusable YAML in the repository, not SQLite.

## Case 087

The request uses two IDs, `pid-query-one` and `pid-query-two`, both matching PID SD-JWT VC `urn:eudi:pid:1` and requesting `given_name`.

Verified evidence:

- Two PID cards appeared in one consent screen.
- Both request accordions exposed `Given Name(s)`.
- Manual sharing succeeded to `EUDI Remote Verifier`.
- Both result accordions exposed the same expected given-name value.
- A fresh verifier transaction returned VP-token entries for both query IDs.

Follow-up: inventory row 087 still describes an earlier `Credential not found` failure. Reconcile it with the later successful evidence and run the final committed Maestro flow completely green after clearing Chrome.

## Cases 088 and 089

- 088: `trusted_authorities` contains `type: unsupported`.
- 089: `trusted_authorities` omits `type`.
- Both now use `mode: request_rejected`; do not revert to `no_match`.
- Emulator 089 returned Home after the malformed request, which is the allowed discontinuation outcome.

Follow-ups:

- Synchronize the emulator-tested 089 leaf flow with the pipeline's inline `action_code`; verify unlock before and after `openLink`.
- Rerun 088 on the emulator after the validator change.
- Review synthetic `json-parse` evidence containing `error: invalid_request`. The observed 089 behavior was discontinuation, not a displayed or captured `invalid_request` response.

## Case 090

OID4VP 6.1.1 defines the required property as plural `values`; the FCAF prose uses singular “value” descriptively. The implemented request keeps valid `type: aki` and omits `values`.

Assertions independently prove that `type` is present, `values` is absent, the request is rejected/discontinued, and visual evidence exists. A clean Maestro run after clearing Chrome returned the wallet to Home without consent or success, satisfying the allowed discontinuation outcome.

## Next candidate

`WS_RP_MS_ProtocolMessages__091`: reject `trusted_authorities.type` when it is not a JSON string. Cover representative invalid JSON types as separate observable cases.

## Parallel ownership

Reserve one test ID per agent. Avoid simultaneous edits to:

- `maestro-preconditions/all-preconditions.yaml`
- `maestro-preconditions/README.md`
- `pkg/fcaf/catalog/loader_test.go`
- `pkg/fcaf/implementation-inventory.csv`

Validate with `go test ./pkg/fcaf/...` and `git diff --check`. Run Maestro on `emulator-5580` with Chrome cleared before claiming conformance behavior.

Active worktrees prepared from `54373c67`:

- 090: `/tmp/credimi-fcaf-090`, branch `test/fcaf-090-missing-authority-value`
- 091: `/tmp/credimi-fcaf-091`, branch `test/fcaf-091-authority-type-format`
- 092: `/tmp/credimi-fcaf-092`, branch `test/fcaf-092-authority-value-format`
- 093: `/tmp/credimi-fcaf-093`, branch `test/fcaf-093-authority-value-items`

Each worktree contains an untracked `AGENT_TASK.md`. Agents own only test-specific artifacts. Shared integration files remain owned by the main worktree and must be updated after the four branches are reviewed/cherry-picked. Code work can run concurrently; emulator/Maestro runs cannot because all agents share `emulator-5580`.
