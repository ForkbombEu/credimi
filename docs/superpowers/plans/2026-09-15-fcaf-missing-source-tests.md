# FCAF Missing Source Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Assess every test in the original relying-party `Unimplemented source tests` section, define each case supported by exact beta Capture evidence, and explicitly block the rest.

**Architecture:** Source-specific test YAML files bind only to their scenario's named Capture evidence. Focused validators decode SD-JWT, mdoc, and compact JWE artifacts; scenario changes expose precisely those artifacts. The backlog records unavailable fixture and transport controls instead of inventing evidence.

**Tech Stack:** Go FCAF validators, YAML source scenarios, Capture Wallet OpenID4VCI/OpenID4VP API, Maestro mobile automation.

**Spec:** `docs/superpowers/specs/2026-09-15-fcaf-missing-source-tests-design.md`

## Global Constraints

- Preserve the generated-pipeline boundary: edit source scenarios/tests and run `make fcaf-generate`; never edit generated pipelines.
- Bind assertions to `pipeline.<source>.outputs.<name>` from the matching scenario only.
- Use a real raw response or decoded presentation for protocol assertions; screenshots only satisfy visual-evidence requirements.
- Do not claim tests needing unsupported beta inputs or fixture types; move them to `Blocked` with the precise missing control.

---

### Task 1: Record unsupported source tests

**Files:**
- Modify: `config_templates/fcaf/wallet_solution/relying_party/ASSERTION_REVIEW_BACKLOG.md`

**Produces:** A zero-ambiguity backlog containing only implementation candidates.

- [x] Move `002d`, `003a`, `003b_UF`, `ProtocolMessages 003_UF`, `RpIntegrity 013b_UF`-`013c_UF`, and `TrustMechanisms 101` variants to `Blocked`.
- [x] State the unavailable Wallet profile, DC API surface, signer control, mdoc status fixture, or WRPAC fixture for each item.

### Task 2: Add exact protocol-artifact validators test first

**Files:**
- Modify: `pkg/fcaf/validators/sdjwt.go`, `pkg/fcaf/validators/generic.go`
- Test: `pkg/fcaf/validators/sdjwt_test.go`, `pkg/fcaf/validators/generic_test.go`

**Produces:** Validators for status-list structure, KB-JWT algorithm/signature/sd_hash, SD-JWT compact components/disclosure digests, vp_token type shape, and JWE protected-header constraints.

- [x] Add focused table-driven tests for new KB-JWT, request-URI, and JWE-header predicates.
- [x] Run focused validator tests before and after implementation.
- [x] Implement only the predicates supported by direct Capture artifacts.

### Task 3: Define Capture scenarios and exact test YAML

**Files:**
- Create: exact `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_*.yaml` files for the 45 supported source IDs.
- Modify: matching source scenarios under `config_templates/fcaf/wallet_solution/relying_party/scenarios/`.

**Produces:** Implemented definitions for engagement, DCQL selection/no-match, request-URI, SD-JWT status/serialization, response structure, device binding, signed RP integrity, session encryption, and disclosure integrity.

- [x] Add each implemented source ID once to its owning scenario and bind every assertion to exact session output.
- [x] Use request/response artifacts for protocol assertions and screenshots only for source-required visual evidence.
- [x] Use exact source paths and standalone YAML IDs, never parent-case aliases.

### Task 4: Regenerate and validate definition integration

**Files:**
- Modify: generated `config_templates/fcaf/wallet_solution/relying_party/pipelines/fcaf-wallet-solution-relying-party-complete-validation.yaml` via generator only.

**Produces:** A generator-valid aggregate owning each new test exactly once.

- [x] Run `make fcaf-generate`.
- [x] Run the focused catalog, generator, and validator suites.
- [x] Run `git diff --check`.
