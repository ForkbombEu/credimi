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
- Catalog count: 182 tests after the uncommitted case 094 implementation.

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

## Case 091

`WS_RP_MS_ProtocolMessages__091` rejects `trusted_authorities.type` when it is not a JSON string. The implemented matrix covers `null`, `true`, `false`, `0`, a non-zero number, array, and object. Every request keeps `values` as a valid non-empty string array.

The dedicated `trusted_authority_property_type` validator proves the nested property is present and has the wrong JSON type, rejects evidence for a missing `type`, requires valid `values` while testing `type`, and fails if the wallet returns a credential. A clean Maestro run after clearing Chrome completed all seven variants. The final UI hierarchy showed Home, so the observed wallet behavior is allowed interaction discontinuation, not a captured `invalid_request` response.

## Case 092

`WS_RP_MS_ProtocolMessages__092` rejects `trusted_authorities.values` when it is not a JSON array. The implementation uses the normative plural property despite the FCAF source's singular `value` wording; that discrepancy is already documented in `TEST-AUTHOR-FEEDBACK.md`.

The matrix covers `null`, `true`, `false`, `0`, a non-zero number, string, and object while preserving `type: aki`. The nested validator requires `values` to be present, verifies its JSON type, requires `type` to remain a non-empty string, and fails if a malformed request returns a credential. A clean Maestro run after clearing Chrome completed all seven variants and produced seven screenshots.

## Case 093

`WS_RP_MS_ProtocolMessages__093` rejects an array-valued `trusted_authorities.values` property when any array item is not a JSON string. The dedicated `trusted_authority_array_item_type` validator requires a non-empty outer array, a valid non-empty string `type`, and at least one item with the invalid type; an all-string array fails the malformed-item assertion. Unit coverage also includes a mixed string plus non-string array.

The matrix covers null, booleans, zero, a non-zero number, nested array, and object items. A clean Maestro run after clearing Chrome completed all seven variants and produced seven screenshots.

## Case 094

`WS_RP_MS_ProtocolMessages__094` rejects a normative `trusted_authorities.values` array containing an empty string. The dedicated `trusted_authority_empty_string_item` validator requires a valid non-empty string `type`, a non-empty array containing only strings, and at least one empty item; non-string items remain case 093 evidence.

The implementation covers a single empty string and a mixed valid-plus-empty array. A clean Maestro run after clearing Chrome completed both variants and produced two screenshots.

## Next candidate

`WS_RP_MS_ProtocolMessages__115` is the next mandatory protocol-message candidate.

## Case 114

114 uses `claim_path_empty` to prove the `path` property is present as an empty array. It distinguishes this from a missing path, `null`, non-array values, and a valid non-empty path. Passing evidence requires no `vp_token` and a real `error: invalid_request`.

The public verifier rejected request creation with HTTP 400 while deserializing `ClaimPath` (`ClaimPath must not be empty`), before the request could reach the Wallet. Device-level execution requires the raw mock-verifier service tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 13. The source expected result contains the typo `invalid request_error`; Issue 15 records the upstream correction to `invalid_request`.

## Case 113

113 uses `claim_path_missing` to prove a claim object omits the `path` property. It explicitly distinguishes absence from present `null`, empty-array, and valid path values. Passing evidence requires no `vp_token` and a real `error: invalid_request`.

The public verifier rejected request creation with HTTP 400 while decoding `ClaimsQuery` because `path` is required, before the request could reach the Wallet. Device-level execution requires the raw mock-verifier service tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 13.

## Case 112

112 uses `invalid_claim_id_characters` to require a present non-empty claim `id` containing at least one character outside ASCII alphanumeric, underscore, and hyphen. Unit evidence covers dot, space, colon, slash, and non-ASCII input, plus the valid boundary `Name_01-test`. Passing evidence requires no `vp_token` and a real `error: invalid_request`.

The public verifier rejected request creation with HTTP 400 in `DCQLId.ensureValid`, before the request could reach the Wallet. Device-level execution requires the raw mock-verifier service tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 13.

## Case 111

111 uses `empty_claim_id` to prove the claim `id` property is present and exactly the empty string, distinguishing it from a missing ID and from non-empty IDs. Passing evidence requires no `vp_token` and an actual `error: invalid_request`; returning Home is not sufficient.

The public verifier rejected request creation with HTTP 400 in `ClaimId` validation (`Value cannot be be empty`), before the request could reach the Wallet. Device-level execution requires the raw mock-verifier service tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 13.

## Case 110

110 uses `duplicate_claim_ids` to require two claims in the same credential query to repeat a non-empty `id`, no `vp_token`, and a real `invalid_request` response. A duplicated ID across separate credential queries is explicitly not treated as this malformed case.

The public verifier rejected the probe with HTTP 400 before request creation: `CredentialQuery.ensureUniqueIds` reported that the same claims ID must not occur more than once. Device-level execution therefore requires the raw mock-verifier service tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 13. The reusable Maestro flow accepts a signed mock-verifier deep link through `DCQL_DUPLICATE_CLAIM_IDS_PRESENTATION_URL`.

The direct by-value emulator probe reached the reference Wallet but displayed no error page; after processing, the Wallet was on Home. This is a failed case 110 result because the source explicitly requires `invalid_request`; discontinuation is not an accepted alternative. The Maestro flow therefore requires an error UI and fails on Home, while the validator independently requires the mock verifier to capture an actual `error: invalid_request` response and rejects evidence that only lacks a `vp_token`.

## Case 109

109 uses a dedicated `claims_without_id_without_claim_sets` validator. It proves every requested claim omits `id`, the credential query omits `claim_sets`, claim paths are valid, and the Wallet returns a presentation under the credential query ID. The pipeline and standalone Maestro flow require the visible successful-sharing state; a parsed request or absence of an error is insufficient.

Reference-wallet verification did not pass. The public verifier accepted two fresh requests with a claim path and no claim `id` or `claim_sets`. On the first run, the post-link `Welcome back` gate discarded the pending interaction; the flow was corrected to unlock before opening the link and to handle the resolver plus any second unlock. On the clean rerun, the Wallet still returned Home without showing `DATA SHARING REQUEST`, and the verifier transaction returned HTTP 400 with an empty body. Keep the strict positive assertions: this is a Wallet failure or an unresolved wallet-core interoperability issue, not grounds to treat discontinuation as acceptance.

## Case 105

105 now has a dedicated `claims_present` validator requiring every credential query to contain a non-empty `claims` array, valid non-empty string paths, and a matching `vp_token`. The verifier accepted a fresh claims-bearing request and Maestro drove the wallet through PIN entry, but the wallet returned Home without showing the consent/share screen. Restarting the wallet process and retrying produced the same result.

TODO: finish 105 emulator diagnosis with runner-accessible wallet logcat and verifier transaction evidence. Direct ADB logcat is currently blocked because the sandbox cannot start the ADB smartsocket daemon; Maestro MCP can still inspect and drive the emulator. The verifier transaction endpoint returned HTTP 400 with an empty body after the wallet interaction.

## Case 106

106 uses a dedicated PID claim path, `claim_that_does_not_exist`, and the `claims_path_no_match` validator proves that the request contains a non-empty claim path and returns no `vp_token`.

Emulator evidence is incomplete: the verifier accepted the request, the Wallet accepted PIN `123456`, and then returned Home without consent or presentation. The verifier transaction endpoint returned HTTP 400 with an empty body. The source expects an observable `access_denied` response describing that no credentials match; current evidence proves only absence of a credential response. Keep this residual gap explicit until verifier diagnostics or protocol evidence are available.

## Case 107

107 uses the valid `given_name` path with a deliberately mismatched `values` constraint. The dedicated `claims_values_no_match` validator requires non-empty `path` and `values` arrays and proves that no `vp_token` is returned.

The emulator accepted the request and PIN, then returned Home without consent or presentation. The verifier transaction endpoint returned HTTP 400 with an empty body. As with 106, this proves no credential was returned but does not prove the source-required `access_denied` response or description.

## Case 108

108 proves the request contains non-empty `claims` and `claim_sets`, at least one claim omits `id`, and no `vp_token` is returned. The public verifier rejected request creation with HTTP 400 `Unknown claim ids` from `ClaimSet.ensureKnownClaimIds`, so the request never reached the Wallet. Device-level execution requires the raw mock-verifier service tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 13.

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
