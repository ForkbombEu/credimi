---
name: fcaf-definitions
description: Implement, review, run, and maintain Credimi FCAF wallet-solution relying-party test definitions, aggregate pipeline scenarios, Maestro flows, validators, and implementation-inventory entries. Use when selecting the next FCAF test, translating an FCAF Markdown scenario into Credimi artifacts, verifying assertions against the reference Android wallet, or coordinating parallel FCAF test work.
---

# FCAF definitions

Implement one FCAF test number end to end. Treat protocol evidence and visual evidence as separate proof obligations.

## Establish context

1. Read repository `AGENTS.md`.
2. Read `pkg/fcaf/MEMORY.md` for current progress, known failures, and the next candidate.
3. When the test uses capture-wallet as issuer or verifier, read
   `pkg/fcaf/CAPTURE_WALLET_API.md` and classify the required session input,
   Wallet delivery, and evidence capture before writing a scenario.
4. Check `git status --short`; preserve unrelated work.
5. Read the exact source test under the sibling FCAF repository. Do not infer the objective from neighboring IDs or the generated inventory.
6. Inspect neighboring implemented tests and validators, but do not copy a pattern until its semantics match.

Source requirements normally live under:

`/home/puria/src/github.com/eu-digital-identity-wallet/eudi-doc-functional-conformance-assessment/main/docs/fcaf/suts/wallet_solution/relying_party/`

## Select work

- Prefer the next unimplemented `EUDI_required` wallet-solution/relying-party row.
- Skip TSL/MTSL, Digital Credentials API, W3C Digital Credentials API, and CAW tests unless the user changes scope.
- Detect duplicate inventory rows before implementing. An older `adapt manual` row may coexist with a newer ready row.
- Announce the test ID before editing. For parallel work, reserve one test number per agent.
- Serialize edits to shared integration files: `all-preconditions.yaml`, `README.md`, `loader_test.go`, and `implementation-inventory.csv`.

## Build the artifact set

Create or update applicable source artifacts under:

`config_templates/fcaf/wallet_solution/relying_party/`

The usual set is:

1. `scenarios/fcaf-wallet-solution-relying-party-<flow>.yaml`
2. `tests/<FCAF-ID>.yaml`
3. `pkg/fcaf/implementation-inventory.csv`

Scenario pipeline must end in `fcaf-validation`, own each selected test exactly once, and expose every test evidence binding under its exact `pipeline.<source>.outputs.<name>` key. Validators consume these aggregate outputs directly. Do not add catalog precondition YAML, assessment workflows, or unrelated-output fallback behavior.

After changing scenarios or tests, run `make fcaf-generate`. This regenerates the sole deployable pipeline:

`pipelines/fcaf-wallet-solution-relying-party-complete-validation.yaml`

Keep inline `action_code` synchronized with emulator-tested behavior. Prefer extracting shared behavior only when runner resolution is deterministic.

## Preserve assertion integrity

- Encode the exact malformed or valid property from the source scenario.
- Do not use `no_match` for malformed-request rejection. Use `request_rejected` when the accepted outcomes are `invalid_request`, an unspecified error, or interaction discontinuation.
- Do not manufacture `error: invalid_request` evidence when the wallet only returned Home. Record the observed discontinuation and seek real verifier/runner evidence where possible.
- A UI screenshot alone does not prove a protocol response. A parsed request alone does not prove wallet behavior.
- For multiplicity, verify both query IDs in the verifier response and both credential entries in the UI.
- For typed properties, cover valid boundaries and representative invalid JSON types. Use duplicated cases when they produce independently useful evidence.
- Match the inventory's required evidence column. If visual evidence is required, add and validate it.

## Run on the reference wallet

Read [emulator.md](references/emulator.md) before the first emulator run in a session.

Use this loop:

1. Clear stale Chrome state while preserving wallet data.
2. Ensure the wallet contains the required PID.
3. Start from a deterministic wallet state with `clearState: false`.
4. Unlock before the request when necessary.
5. Open the authorization deep link.
6. Handle a second `Welcome back` PIN screen after `openLink`.
7. Inspect the native hierarchy when selectors fail.
8. Capture screenshots at the exact consent, rejection, and success states required by the test.
9. Query the verifier transaction when a verifier is used.
10. Rerun from scratch after fixing state handling.

Treat returning to `Home` without consent or success as discontinuation only when the source test explicitly allows it.

## Validate

Run the narrow checks while iterating:

```sh
gofmt -w pkg/fcaf/validators/dcql.go  # only when this Go file changed
make fcaf-generate
go test ./cmd/fcaf-pipeline-gen ./pkg/fcaf/... ./pkg/internal/pipeline
git diff --check
```

Before claiming emulator success, require a green Maestro run or explicitly report the exact remaining failed command. Before committing, let repository hooks validate YAML and REUSE metadata.

## Publish only on request

Do not commit or push without explicit user authorization. Follow the exact commit body contract in `AGENTS.md`. This worktree may be detached; the established push target is:

```sh
git push origin HEAD:feat/fcaf-test
```

After completion, update `pkg/fcaf/MEMORY.md` with the commit, evidence, residual risk, and next candidate.
