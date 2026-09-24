# FCAF pending assertions implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the eight tests in the relying-party assertion backlog's `Pending` section either source-correct with direct beta Capture Wallet evidence or explicitly blocked by a newly observed fixture/evidence gap.

**Architecture:** Split the current generic encoding flow into source-specific JSON and mdoc Capture sessions where their request, response, and expected outcome differ. Add narrow validators that inspect the signed request plus the decoded SD-JWT or mdoc response. For issuer integrity, extend certificate-chain validation to exclude one explicit trust-anchor fingerprint, with that fingerprint obtained outside the presented `x5c` chain.

**Tech Stack:** Go FCAF validators and table-driven tests, YAML source scenarios/test definitions, Capture Wallet OpenID4VP sessions, Maestro mobile automation, SD-JWT VC, ISO mdoc/CBOR, X.509.

**Spec:** `docs/superpowers/specs/2026-09-15-fcaf-pending-assertions-design.md`

## Global constraints

- Edit source scenarios and test definitions only; regenerate pipelines with `make fcaf-generate` and never hand-edit generated pipelines.
- Bind each assertion to the matching scenario's exact `pipeline.<source>.outputs.<name>` evidence.
- Keep `openid4vp://` for generic OID4VP scenarios unless a source requires a different scheme.
- A request, response, error, certificate, or visual assertion must be evidenced by its own artifact; no unrelated successful session is fallback evidence.
- Do not commit or push unless the maintainer explicitly authorizes it.

---

### Task 1: Qualify the exact Capture fixtures before changing definitions

**Files:**
- Modify: `pkg/fcaf/CAPTURE_WALLET_API.md` only when a probe establishes a new stable capability or limitation.
- Modify: `config_templates/fcaf/wallet_solution/relying_party/ASSERTION_REVIEW_BACKLOG.md` only for a case proven unavailable by this task.

**Produces:** A dated evidence record for every prerequisite and a decision for each case that prevents substitute-fixture assertions.

- [ ] **Step 1: Probe JSON array-selection fixture availability.**

  Create two `openid4vp://` Capture sessions whose signed requests contain,
  respectively, `claims.path: ["degrees", null, "type"]` and
  `claims.path: ["degrees", null, 1]`. Use a Wallet provisioned with the
  source fixtures: one degree object with `type` and one without for 008; one
  one-element array and one two-element array for 011. Fetch the session after
  sharing and preserve only the session IDs and decoded artifact facts in the
  engineering record.

  Expected evidence: each `authorization_request.dcql_query` preserves the
  exact path, and the `vp_token` exposes only the retained value
  (`Bachelor of Science` for 008; `Ph.D.` for 011).

- [ ] **Step 2: Probe the mdoc namespace/element fixture.**

  Obtain the reference Wallet mdoc and create a session requesting
  `format: mso_mdoc`, `doctype_value: eu.europa.ec.eudi.pid.1`, and
  `path: ["org.iso.18013.5.1", "first_name"]`. Decode the returned mdoc when
  it exists; record whether that exact namespace/element is present as CBOR
  text. Repeat with the malformed one-component path, non-string component,
  and absent element `Bob` to establish the captured error shape for 018-020.

  Expected evidence: a positive mdoc for 017/021 and a captured Wallet error
  with no `vp_token` for 018-020. If an exact fixture is absent, do not rename
  it to `given_name` or another PID namespace.

- [ ] **Step 3: Qualify the issuer trust anchor independently of `x5c`.**

  Run a successful SD-JWT Capture presentation. Decode all `x5c` DER entries,
  retrieve the issuer's published trust material or a documented issuer-chain
  source, and compute SHA-256 fingerprints. Establish exactly one trust-anchor
  fingerprint that is not inferred from the presented chain.

  Expected evidence: issuer leaf/intermediate certificate fingerprints and an
  independently sourced anchor fingerprint. A missing anchor source means
  case 014 cannot assert trust-anchor exclusion.

- [ ] **Step 4: Classify any failed prerequisite truthfully.**

  For each failed probe, move that exact ID from `Pending` to `Blocked`, cite
  the missing fixture or artifact (for example, no provisionable `degrees`
  credential or no independently identified trust anchor), and retain the
  raw-observation date. Leave successful IDs in `Pending` for the following
  implementation tasks.

### Task 2: Add JSON claim-path selection assertions for 008 and 011

**Files:**
- Create: `pkg/fcaf/validators/dcql_claim_path_selection.go`
- Create: `pkg/fcaf/validators/dcql_claim_path_selection_test.go`
- Modify: `pkg/fcaf/validators/registry.go`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/scenarios/fcaf-wallet-solution-relying-party-dcql-encoding.yaml` or create dedicated source scenarios when the fixture cannot coexist safely.
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SH_Encoding_TextualEncoding_008.yaml`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SH_Encoding_TextualEncoding_011.yaml`

**Consumes:** Qualified positive SD-JWT sessions from Task 1.

**Produces:** `dcql.claim_path_selection` validator with parameters `expected_claim_path`, `required_values`, and `forbidden_values`; each test consumes its own request/response capture.

- [ ] **Step 1: Write table-driven validator tests first.**

  Add cases for a correct 008 response retaining `Bachelor of Science` and
  omitting a missing `type`; a correct 011 response retaining `Ph.D.` and
  omitting the out-of-range array; wrong request path; missing `vp_token`;
  forbidden value present; and required value absent. The fixture must contain
  both candidate elements so a passing case proves filtering rather than a
  one-element credential.

  Run: `go test ./pkg/fcaf/validators -run TestDCQLClaimPathSelection -count=1`

  Expected: failure until the validator is registered and implemented.

- [ ] **Step 2: Implement the minimal selection validator.**

  Decode the captured authorization request, require exactly the supplied path
  in the matching credential query, parse every SD-JWT presentation belonging
  to that query, and compare disclosed values against `required_values` and
  `forbidden_values`. Fail if no matching SD-JWT presentation exists; do not
  inspect a screenshot as a substitute.

- [ ] **Step 3: Make cases 008 and 011 source-specific.**

  Replace `sdjwt.claim_utf8_string` with `dcql.claim_path_selection`; bind
  each test to the scenario that creates its exact path and fixture. Keep a
  visual assertion only if the source inventory requires visual proof. Remove
  `given_name` from these scenarios unless it is independently part of the
  source query.

- [ ] **Step 4: Run unit and catalog checks.**

  Run: `go test ./pkg/fcaf/validators ./pkg/fcaf/catalog -count=1`

  Expected: all selection success and rejection cases pass, and catalog loading
  resolves the new validator IDs and evidence bindings.

### Task 3: Implement exact mdoc path assertions for 017 through 021

**Files:**
- Modify: `pkg/fcaf/validators/dcql.go`
- Modify: `pkg/fcaf/validators/dcql_test.go`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/scenarios/fcaf-wallet-solution-relying-party-pid-mdoc-data-model.yaml` or create focused `mdoc-textual-encoding` scenarios.
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SH_Encoding_TextualEncoding_017.yaml`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SH_Encoding_TextualEncoding_018.yaml`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SH_Encoding_TextualEncoding_019.yaml`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SH_Encoding_TextualEncoding_020.yaml`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SH_Encoding_TextualEncoding_021.yaml`

**Consumes:** Task 1 mdoc fixture and captured error shape.

**Produces:** Exact positive mdoc element and negative mdoc-path evidence, with each malformed path distinguishable in the signed request.

- [ ] **Step 1: Add failing unit tests for mdoc exactness.**

  Extend `dcql_test.go` with table cases requiring that
  `mdoc_claim_path_presentation` rejects a response whose query path is wrong,
  whose selected element is absent, or whose element is not a CBOR text value.
  Add a negative mode, `mdoc_claim_path_error`, that accepts only an exact
  request path plus captured Wallet error and no `vp_token`; cover wrong path,
  response-present, and error-absent failures.

  Run: `go test ./pkg/fcaf/validators -run 'TestDCQLResponseConstraintsValidator.*MDoc' -count=1`

  Expected: failure until the mode and CBOR text check are implemented.

- [ ] **Step 2: Implement the positive and negative mdoc predicates.**

  Tighten `mdoc_claim_path_presentation` to require the two string path
  components, one matching `mso_mdoc` query, a returned presentation for that
  query ID, and a parsed mdoc element whose CBOR major type is text. Implement
  `mdoc_claim_path_error` to preserve malformed paths without validating them
  as a normal mdoc path, require the documented captured error, and reject a
  `vp_token`.

- [ ] **Step 3: Define one exact request per source case.**

  Use `["org.iso.18013.5.1", "first_name"]` for 017 and 021; each must prove
  the returned `first_name` is CBOR text. Use `["org.iso.18013.5.1"]` for
  018, `["org.iso.18013.5.1", 123]` for 019, and
  `["org.iso.18013.5.1", "Bob"]` for 020; each must use
  `mdoc_claim_path_error`. Do not attach these cases to the all-claims
  `given_name` scenario unless Task 1 proves the exact source element there.

- [ ] **Step 4: Verify each request and device outcome.**

  Run `make fcaf-generate`, execute every changed dedicated scenario on the
  reference Wallet, fetch its Capture session, and inspect the decoded request
  and response/error before marking the test implemented. Then run:

  `go test ./pkg/fcaf/validators ./pkg/fcaf/catalog ./cmd/fcaf-pipeline-gen -count=1`

### Task 4: Prove trust-anchor exclusion for IssuerIntegrity 014

**Files:**
- Create: `pkg/fcaf/validators/sdjwt_issuer_x5c_excludes_anchor.go`
- Create: `pkg/fcaf/validators/sdjwt_issuer_x5c_excludes_anchor_test.go`
- Modify: `pkg/fcaf/validators/registry.go`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/scenarios/fcaf-wallet-solution-relying-party-dcql-issuer-integrity.yaml`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/tests/WS_RP_SM_IssuerIntegrity__014.yaml`

**Consumes:** Task 1's independently sourced trust-anchor SHA-256 fingerprint.

**Produces:** `sdjwt.issuer_x5c_excludes_trust_anchor`, parameterized by `trust_anchor_sha256`, plus a source-specific SD-JWT presentation output.

- [ ] **Step 1: Write failing certificate-chain tests.**

  Build in-memory valid DER leaf, intermediate, and root certificates. Test a
  passing `[leaf, intermediate]` chain; failure when the root is included;
  failure for an empty/malformed `x5c`; failure when the fingerprint parameter
  is malformed; and failure when the SD-JWT presentation cannot be parsed.

  Run: `go test ./pkg/fcaf/validators -run TestSDJWTIssuerX5CExcludesTrustAnchor -count=1`

- [ ] **Step 2: Implement and register the validator.**

  Parse the SD-JWT issuer protected header, require a non-empty valid DER
  `x5c` array, compute SHA-256 over every certificate DER value, and fail when
  any equals the supplied lowercase-hex trust-anchor fingerprint. Report the
  chain count and matching index on failure. Do not treat the final certificate
  as an anchor without the parameter.

- [ ] **Step 3: Bind 014 to its actual SD-JWT evidence.**

  Make the issuer-integrity scenario request `dc+sd-jwt`, expose the matching
  `vp_token` as `pid_sdjwt`, and replace generic `dcql_valid` with both the
  format/presentation assertion and `sdjwt.issuer_x5c_excludes_trust_anchor`.
  The test YAML's fingerprint parameter must come from the Task 1 documented
  trust source, not a value copied out of `x5c`.

- [ ] **Step 4: Verify on a new Capture session.**

  Run the scenario on the reference Wallet, decode the response manually once,
  and confirm the captured x5c has issuer-chain certificates but no fingerprint
  matching the independently obtained anchor. Run:

  `go test ./pkg/fcaf/validators ./pkg/fcaf/catalog -count=1`

### Task 5: Reconcile the backlog and generated aggregate

**Files:**
- Modify: `config_templates/fcaf/wallet_solution/relying_party/ASSERTION_REVIEW_BACKLOG.md`
- Modify: `config_templates/fcaf/wallet_solution/relying_party/pipelines/fcaf-wallet-solution-relying-party-complete-validation.yaml` through the generator.
- Modify: `pkg/fcaf/MEMORY.md`

**Consumes:** Completed direct evidence or precise prerequisite failures from Tasks 1-4.

**Produces:** No stale `Pending` definition: every ID is implemented with exact evidence or has a durable, specific Blocked reason.

- [ ] **Step 1: Update each backlog row after evidence review.**

  Move successfully corrected IDs out of `Pending` and record their scenario
  plus validator. Move failed prerequisites to `Blocked` with the exact absent
  fixture, response artifact, or trust-anchor source and the probe date.

- [ ] **Step 2: Regenerate and run integration validation.**

  Run:

  ```sh
  make fcaf-generate
  go test ./pkg/fcaf/catalog ./cmd/fcaf-pipeline-gen ./pkg/fcaf/validators -count=1
  git diff --check
  ```

  Expected: every test has one source-scenario owner, generated pipeline
  references resolve, and no whitespace errors remain.

- [ ] **Step 3: Record durable progress.**

  Update `pkg/fcaf/MEMORY.md` with the evidence sources, device-run result,
  remaining blocked reason if any, and the next candidate. Commit only after
  explicit maintainer authorization.
