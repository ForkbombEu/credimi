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

## Aggregate validation architecture

On 27/08/2026, maintainer removed legacy FCAF orchestration. `/api/fcaf/run`, assessment/precondition workflows, catalog precondition definitions, and `fcaf run --tests-file` must stay removed. Validators consume exact named aggregate pipeline outputs directly.

Maintain 112 evidence-producing source definitions under `config_templates/fcaf/wallet_solution/relying_party/scenarios/`. `make fcaf-generate` combines them into one deployable pipeline, prefixes scenario step IDs, continues after scenario failures, merges 115 exact evidence sources, and runs one final `fcaf-validation` over all 559 tests. `pipelines/` contains only this generated complete-validation pipeline, so sync/run creates one top-level FCAF execution.

## Happy flow aggregate pipeline

On 31/08/2026 `cmd/fcaf-pipeline-gen` gained a third generated pipeline,
`fcaf-wallet-solution-relying-party-happy-flow-validation.yaml`: the
shared-evidence positive batch of the FCAF catalog, deliberately NOT the
complete assessment. `happyFlowScenarioNames` selects the 14 scenarios that
own every positive test batch (>= 5 positive tests each): 423 test IDs, 17
evidence sources, 53 steps, 21 mobile-automation actions. Excluded: all
negative tests and the fragmented one-test-per-interaction DCQL tail that
only the complete validation aggregate covers. Reuses only existing wallet
actions (`onboarding-1`, `getcredential-generic-credential-without-authentication`,
`fcaf-engagement-haip-vp`); no new Maestro actions are needed. The
maintainer rejected both a positives-only aggregate of all 75 positive
scenarios (500 tests, ~86 wallet actions, "full assessment minus negatives")
and a strict single-presentation flow (72 tests, below the 200+ check
expectation). Deployment to the fcaf-1 org on credimi.io: apply the CLI org
rewrite (`forkbomb-bv-andrea/` -> `fcaf-1/`), drop `runtime.global_device_id`,
and upload the record; the webapp queue flow injects the UI-selected device
as `global_device_id` at queue time. `installed_from_external_source` is a
platform sentinel (use the wallet pre-installed on the device host), not a
wallet_versions lookup, and needs no org rewrite.

## Happy flow fcaf-1 deployment incident

The first credimi.io/fcaf-1 run of the happy flow pipeline (31/08/2026,
17:21) failed during workflow setup: `markExternalInstallSteps` resolved
`workflowengine.AsString(nil)` — the literal string `"<nil>"` — as an
action identifier for the eleven `installed_from_external_source` steps
that carry inline `action_code` and no `action_id`, so the canonify
validate endpoint returned CRE310 `invalid path "<nil>"`. Two-part remedy:

1. Repo fix (uncommitted in the fcaf-happy-flow worktree):
   `markExternalInstallSteps` now skips sentinel steps without a string
   `action_id`; regression test
   `TestMarkExternalInstallStepsSkipsInlineActionCodeSteps` fails on the
   unfixed code. Deploy with the next server release.
2. Instance-side pipeline shim (record `7j70w1pf7pqpl23`, patched 17:27):
   those eleven steps now declare
   `action_id: fcaf-1/eudiw-beta-wallet/fcaf-engagement-haip-vp`
   (category `verify-credential`, never `install-app`). Execution is
   unchanged because the mobile-automation child workflow runs
   `payload.ActionCode`, not the stored action; the shim only feeds the
   category lookup. Replace the shim with the repo fix once deployed.

The complete-validation pipeline hits the same setup bug on any current
server build; it is not fcaf-1-specific.

Every test has exactly one scenario owner. Do not restore sole-output fallback: one wallet interaction must never stand in for incompatible scenarios. A complete run may still contain many sequential wallet interactions. Mock-verifier-blocked tests must report blocked/failed from missing real evidence, never synthetic conformance passes.

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

## TextualEncoding 004

`WS_RP_SH_Encoding_TextualEncoding_004` uses a dedicated Capture Wallet DCQL
scenario with the syntactically valid but reversed path
`["street_address", "address"]`. Capture's PID contains the contrasting valid
`address.street_address` claim, so the no-match and `access_denied` assertions
test left-to-right processing rather than merely a missing claim. The scenario
reuses the established no-match Wallet UI flow and retains exact session and
visual evidence.

## TextualEncoding 005

`WS_RP_SH_Encoding_TextualEncoding_005` reuses the existing Capture Wallet
DCQL request for the top-level `given_name` path. The claims-subset validator
requires that exact requested path to be disclosed and `family_name` to remain
absent, proving selective disclosure of a named top-level attribute.

## TextualEncoding 006

`WS_RP_SH_Encoding_TextualEncoding_006` uses a dedicated Capture Wallet query
for top-level `street_address`. Capture's PID has `address.street_address`,
but no root `street_address`, so the test distinguishes a non-matching
top-level claim pointer from a missing fixture. The source accepts a captured
error or interaction discontinuation; the validator rejects any presentation.

## TextualEncoding 007

`WS_RP_SH_Encoding_TextualEncoding_007` uses the dedicated Capture Wallet path
`["given_name", "firstname"]`. Capture's standard PID exposes `given_name` as a
scalar, so applying the second member distinguishes a traversal-type error from
an absent claim. The `wallet_error_required` validator rejects both a
presentation and a silent discontinuation, requiring the protocol error stated
by the source.

## TextualEncoding 009

`WS_RP_SH_Encoding_TextualEncoding_009` uses the dedicated Capture Wallet path
`["address", null, "street_address"]`. Capture's standard PID exposes
`address` as an object with `street_address`, so the null selector exercises
the required non-array failure rather than a missing claim. It reuses the
strict `wallet_error_required` validator to reject both a presentation and a
silent discontinuation.

## TextualEncoding 010

`WS_RP_SH_Encoding_TextualEncoding_010` uses the dedicated Capture Wallet path
`["address", "street_address", 0]`. Capture's standard PID exposes
`address.street_address` as a scalar, so the final integer selector exercises
the required non-array failure. The source example uses a conflicting null
component, but its objective and expected result require a non-negative integer
component; the scenario follows those normative statements and requires an
error without a presentation.

## TextualEncoding 012

`WS_RP_SH_Encoding_TextualEncoding_012` uses the dedicated Capture Wallet path
`["address", "street_address", false]`. The Boolean is an unsupported DCQL
claim-path component, so the Wallet must reject the request before it can
produce a presentation. The scenario asserts the exact component type and
requires an error without a presentation.

## TextualEncoding 013

`WS_RP_SH_Encoding_TextualEncoding_013` uses the dedicated Capture Wallet path
`["address", "unavailable_address_member"]`. It first selects the known PID
address object and then produces an empty selection because the second member
does not exist. The scenario asserts that exact path and requires an error
without a presentation.

## TextualEncoding 014

`WS_RP_SH_Encoding_TextualEncoding_014` reuses the established PID mdoc
presentation flow. Its Capture request includes the valid two-element claim
path `["eu.europa.ec.eudi.pid.1", "given_name"]`; the dedicated DCQL validator
requires that exact path and a returned mdoc containing the element. The mdoc
UTF-8 validator independently proves the selected element is CBOR text.

## TextualEncoding 015

`WS_RP_SH_Encoding_TextualEncoding_015` reuses the same PID mdoc exchange as
case 014. The DCQL assertion proves the first component of the exact path
`["eu.europa.ec.eudi.pid.1", "given_name"]` resolves as the namespace in the
returned mdoc; the mdoc validator separately confirms CBOR UTF-8 encoding.

## TextualEncoding 016

`WS_RP_SH_Encoding_TextualEncoding_016` uses a dedicated PID mdoc flow. Capture
accepted and preserved its absent namespace path
`["org.iso.18013.5.1", "first_name"]` in session
`13aa1df4-e5b8-432f-b208-5454d71bbea0`. The test requires that exact mdoc
query plus a Wallet error and no `vp_token`; it does not accept silent
discontinuation.

## Next candidate

## Pending assertion review (15/09/2026)

`WS_RP_SH_Encoding_TextualEncoding_018` through `020` now have isolated mdoc
Capture scenarios. Their `mdoc_claim_path_error` assertion proves the exact
one-component, non-string-component, or absent-element path; it also requires
a captured Wallet error and rejects any `vp_token`. The paths use the actual
Capture PID namespace `eu.europa.ec.eudi.pid.1`, so 020 isolates a missing data
element instead of accidentally testing a missing namespace.

`008`, `011`, `017`, `021`, and `IssuerIntegrity__014` were moved to Blocked:
the public beta service cannot provision the source `degrees` fixture or the
positive `org.iso.18013.5.1.first_name` mdoc fixture, and exposes no
independently identified trust-anchor certificate needed to prove its absence
from `x5c`. The signed-request probe `c12f2511-84cf-40db-8212-d6e0b1284fae`
did confirm that Capture preserves the relevant request paths; it does not
create the missing Wallet fixture or response evidence.

`WS_RP_SM_DeviceBinding__008` is the next runnable mandatory candidate. Case
119 duplicates case 114; cases 124-146 and 153-159 are intentionally skipped
where the required raw request, transaction-data fixture, or configurable
verifier response cannot be produced by the public service.

## Case 123

Capture Wallet accepts the source-defined `path: [true]` malformed
claim-path member and preserves it unchanged in its signed Authorization
Request. The dedicated scenario therefore exercises the Wallet directly.
Its strict `invalid_request_required` validator requires a captured
`invalid_request` and rejects both a presentation and a silent discontinuation.
The generated aggregate pipeline was refreshed; an emulator run remains needed
to establish the reference Wallet's conformance result.

## Case 122

Capture Wallet accepts the source-defined non-array `path: "given_name"` and
preserves it in the signed Authorization Request. The scenario therefore reuses
the strict `invalid_request_required` validator from case 123, requiring a
captured error and no presentation. An emulator run remains needed to establish
the reference Wallet's conformance result.

## Case 121

Capture Wallet accepts the source-defined `path: [-1]` and preserves it in the
signed Authorization Request. The scenario reuses the strict
`invalid_request_required` validator, requiring a captured error and no
presentation. An emulator run remains needed to establish the reference
Wallet's conformance result.

## Case 120

Case 120 uses the same source-defined `path: [true]` value as case 123. Its
dedicated scenario reuses the verified Capture Wallet delivery path and strict
`invalid_request_required` validator, requiring a captured error and no
presentation. An emulator run remains needed to establish the reference
Wallet's conformance result.

## Case 115

The dedicated non-array claim-path scenario sends `path: "given_name"`, which
Capture Wallet preserves in its signed Authorization Request. Its
`claim_path_non_array` validator verifies the malformed structure and requires
the captured `invalid_request`; a presentation or a silent discontinuation
fails. An emulator run remains needed to establish the reference Wallet's
conformance result.

## Case 114

The dedicated empty claim-path scenario sends the source-defined `path: []`,
which Capture Wallet preserves in the signed Authorization Request. Its
`claim_path_empty` validator verifies the malformed structure and requires the
captured `invalid_request`; a presentation or a silent discontinuation fails.
An emulator run remains needed to establish the reference Wallet's conformance
result.

## Case 113

The dedicated missing claim-path scenario sends the source-defined claim object
without `path`, which Capture Wallet preserves in the signed Authorization
Request. Its `claim_path_missing` validator verifies the malformed structure
and requires the captured `invalid_request`; a presentation or a silent
discontinuation fails. An emulator run remains needed to establish the
reference Wallet's conformance result.

## Case 112

The dedicated invalid-claim-ID scenario sends `claim with spaces!`, which
Capture Wallet preserves in the signed Authorization Request. Its
`invalid_claim_id_characters` validator verifies the malformed ID and requires
the captured `invalid_request`; a presentation or a silent discontinuation
fails. An emulator run remains needed to establish the reference Wallet's
conformance result.

## Case 111

The dedicated empty-claim-ID scenario sends `id: ""`, which Capture Wallet
preserves in the signed Authorization Request. Its `empty_claim_id` validator
verifies the malformed ID and requires the captured `invalid_request`; a
presentation or a silent discontinuation fails. An emulator run remains needed
to establish the reference Wallet's conformance result.

## Mock-verifier skip queue

Do not implement the following negative cases with the public reference
verifier. Keep their inventory status at `missing` until a mock service can
deliver the required request and capture the Wallet's actual protocol result:

- 124: the public endpoint accepts an unknown field in its presentation-create
  JSON but strips it from the signed Authorization Request. A live probe on
  15/07/2026 confirmed `fcaf_unknown_parameter` was absent from the JWT.
- 125-126: the verifier response endpoint must deliberately return either HTTP
  200 with a non-JSON body or HTTP 400 with JSON after receiving the Wallet's
  response.
- 127-128: the response endpoint must return JSON containing an unknown
  parameter, and case 128 also needs an unknown signed request parameter.
- 129-132: the public result API returns only the decrypted Wallet response and
  does not expose the compact JWE. Therefore `kid`, explicit/default `enc`, and
  the original JWT payload structure cannot be asserted.
- 133-134: SUPERSEDED. `raw.presentation_response_http` now exposes the
  Wallet-facing method, headers, and exact body, and both cases are implemented
  on that evidence (see "Case 134" and "Response-transport assertion scope").
- 135: the source does not define a transaction-data type/fixture the Wallet is
  expected to support. Wallet core 0.28.1 explicitly rejects every non-empty
  `transaction_data`, so inventing a type would test case 136 instead.
- 136: the verifier must issue unsupported `transaction_data` and capture the
  Wallet error without opening credential selection.
- 137-140: SUPERSEDED. `presentation_request.scope` reaches the signed request
  and `observed.wallet_response.value.error` captures the Wallet error code, so
  all four are implemented (see "Cases 137, 138, 139, 140, 142, 150").
- 141-145: conflicting query/scope, missing query instructions, unsupported or
  insecure client identifiers, and conflicting stored client metadata all need
  custom signed requests plus exact Wallet error capture.
- 146: the trusted-registry and locally stored verifier metadata state needed
  to trigger `invalid_client` requires a stateful mock verifier/registry.
- 150: SUPERSEDED. Re-probed live on 16/09/2026: Capture accepts and preserves
  `format: vc+sd-jwt` in the signed request, so 150 is implemented and awaits a
  reference-Wallet run, not a mock verifier.
- 153-159: all require signed requests containing unsupported or malformed
  `transaction_data` and exact Wallet error capture. Cases 154-157 also require
  a supported transaction-data schema that the suite does not define.
- DeviceBinding 002-006: the public verifier request object contains no
  `verifier_info` attestation. These cases require a generator for valid,
  malformed, invalid-proof, and unknown-type Verifier Info attestations.

These are implementation skips, not conformance passes or accepted
discontinuations. See `TEST-AUTHOR-FEEDBACK.md` Issues 13 and 19.

## Case 147

147 reuses `pipeline.dcql.no-matching-credentials`, which requests the
deliberately unavailable VCT `urn:credimi:fcaf:no-matching-test-credential`.
The tightened mobile flow clears Chrome, unlocks before opening the request,
requires the Wallet's unavailable-document screen, proves no requested-document
row exists, captures visual evidence, and only then selects `Go Back`. The protocol
assertions require no returned credential and an error value exactly equal to
`access_denied`; Home or a screenshot alone cannot pass the case.

The 15/07/2026 emulator run passed the UI portion: the reference Wallet showed
`The requested document is not available in your EUDI Wallet`, rendered no
credential row, left the Share control disabled, and allowed `Go Back`. It did
not submit an error response. The public
verifier poll returned HTTP 400 with an empty body because the transaction was
not in Submitted state. The reference Wallet therefore fails case 147; do not
weaken the exact `access_denied` assertion or treat the local error screen as a
protocol response.

## Case 148

148 uses a dedicated valid PID request and requires the Wallet to reach the
consent screen before the user explicitly denies consent. In wallet version
2026.06.38, the denial control has accessibility label `Go Back`; there is no
visible `Cancel` label and no confirmation dialog. Visual evidence is captured
both before and after denial. Protocol evidence must omit `vp_token` and contain
`error` exactly equal to `access_denied`; returning Home without a submitted
verifier response cannot pass.

The 15/07/2026 Maestro run passed the UI portion on `emulator-5554`: a valid PID
request reached `DATA SHARING REQUEST`, the first PID accordion exposed `Given
Name(s)`, and `Go Back` returned the Wallet to Home. The Wallet sent no error
response. Polling that exact verifier transaction returned HTTP 400 with an
empty body. The reference Wallet therefore fails case 148; keep the strict
protocol assertions.

## Case 149

149 uses a dedicated valid PID request, selects `Share`, and submits one known
invalid PIN (`111111`) at the transaction-authentication screen. The flow
requires the wallet's explicit `Invalid pin` state and captures it before any
cleanup. It then returns through the request screen to Home so later pipelines
start deterministically. Protocol evidence must omit `vp_token` and contain
`error` exactly equal to `access_denied`.

The 15/07/2026 reusable Maestro flow passed the UI portion on `emulator-5554`: the
Wallet displayed the requested PID and `Given Name(s)`, reached the PIN screen
after `Share`, and displayed `Invalid pin` after one failed attempt. It sent no
error response; polling the same verifier transaction returned HTTP 400 with
an empty body. This is partial evidence, not a conclusive case 149 execution:
one invalid PIN is a failed attempt while the authentication interaction still
allows retries. Completion needs a verifier web-form/manual flow that reaches a
defined terminal authentication failure and exposes the submitted authorization
error. The upstream scenario does not define whether one invalid attempt, terminal lockout,
biometric failure, or cancellation constitutes failed authentication; this is
tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 21.

## Cases 150 and 151

150 is implemented, not mock-verifier blocked. The 15/07/2026 probe that
rejected `format: vc+sd-jwt` with HTTP 400 `{"error":"UnsupportedFormat"}`
described an older verifier. A 16/09/2026 beta probe accepted the format and
preserved it in the signed Request Object, so the Wallet does receive the
request; whether it answers `vp_formats_not_supported` for the pre-final
`vc+sd-jwt` alias is a live reference-Wallet question.

151 is not executable against the reference Wallet because its prerequisite
requires a Wallet that supports `vc+sd-jwt` but does not support `mso_mdoc`.
The reference Wallet supports `mso_mdoc`, so changing the requested document
type would test credential availability rather than format support.

## Case 152

152 starts from a valid public-verifier request URI and adds the deliberately
invalid `request_uri_method=DELETE` authorization parameter. The Maestro flow
requires the Wallet's generic error page, proves that neither `DATA SHARING
REQUEST` nor a requested-document row appears, captures visual evidence, and
returns to Home. Protocol assertions require no `vp_token` and error exactly
equal to `invalid_request_uri_method`.

The 15/07/2026 reusable Maestro flow passed and reached the generic `Oups!
Something went wrong` page, but the Wallet sent no error response. Polling the same verifier
transaction returned HTTP 400 with an empty body. The reference Wallet fails
case 152; do not treat the local generic error page as protocol evidence.

## Cases 153-159

These seven `invalid_transaction_data` cases are not runnable with the public
verifier. They require controlled signed request objects and exact Wallet error
capture. 153 is the exact-error specialization of blocked case 136. Cases
154-157 additionally require a known supported transaction-data type, schema,
field types, ranges, and mandatory fields; no such fixture is defined. Cases
158-159 require controlled `credential_ids` references. Keep them missing until
the mock verifier and transaction-data fixture tracked in Issues 13 and 20 are
available.

## DeviceBinding cases 002-006

A live public-verifier request-object probe on 15/07/2026 contained the normal
x5c-signed authorization request but no `verifier_info` attestation. Cases
002-006 need a controllable Verifier Info attestation generator and cannot be
adapted from that request without changing the signed object. They are tracked
as mock-verifier skips and in `TEST-AUTHOR-FEEDBACK.md` Issue 23.

## DeviceBinding case 007

007 reuses the successful PID SD-JWT all-claims presentation and validates the
issuer-signed `cnf` claim with `sdjwt.cnf_conforms`. The structural validator requires a
non-empty object in the issuer payload, exactly one RFC 7800 proof-of-possession
key representation, valid public EC/RSA/OKP JWK members without private key
material, unpadded base64url key values with curve-specific lengths, a compact
JWE, a non-empty key identifier, or an HTTPS JWK Set URL. Unknown members are
ignored only when a supported confirmation method remains present.

The mandatory `sdjwt.key_binding_matches_cnf` assertion preserves the exact
SD-JWT and compact KB-JWT bytes, verifies the KB-JWT signature with `cnf.jwk`,
enforces `typ: kb+jwt`, rejects unsecured or key-incompatible algorithms,
requires valid `iat`, `aud`, `nonce`, and `sd_hash` claims, and recomputes the
RFC 9901 `sd_hash` over the presented SD-JWT and selected disclosures. Wrong
keys, altered signatures, malformed mandatory claims, algorithm confusion, and
mismatched presentation hashes are covered by focused negative tests.

The shared precondition exposes both the existing singular `pid_sdjwt` output
for older tests and a `pid_sdjwt_presentations` collection for 007. Both 007
validators iterate the full collection and pass only when every returned
credential presentation passes; a failure or unresolved confirmation method is
reported with its presentation index. This prevents the first `query_0` member
from hiding a bad later credential.

Resolver note: `kid`, `jku`, and `jwe` remain valid structural RFC 7800
confirmation methods, but their cryptographic verification needs trusted
external key-resolution or decryption evidence. The cryptographic validator
returns `blocked`, never `pass`, when `cnf.jwk` is absent. A future resolver must
provide the resolved Holder key as evidence; network access must not be hidden
inside the pure validator.

Visual presentation evidence is exposed from the shared PID precondition. A
live reference-wallet run on 15/07/2026 left both stored PID credentials selected
(Filippo and the FCAF test user), displayed their requested claims, and sent two
SD-JWT presentations. Maestro completed the consent and success flow. The exact
verifier response was decoded as a two-member collection; each presentation
contained a distinct EC P-256 `cnf.jwk`, and both passed the structural check,
KB-JWT signature verification, and RFC 9901 `sd_hash` recomputation.

## DeviceBinding case 008

008 reuses the live `pipeline.dcql.holder-binding-type-boundary` valid-true
execution. Its test asserts the requested credential property
`require_cryptographic_holder_binding` equals the boolean `true` exactly, not
merely that it has boolean JSON type. The generic DCQL validator gained
`property_equals` with positive and false-value regression cases for this
purpose. The precondition decodes every returned SD-JWT from
`vp_token.pididentity01`, and 008 applies `sdjwt.cnf_conforms` to all of them.

On 15/07/2026 the public-verifier request with that exact property completed
through Maestro consent and success on `emulator-5554`. The verifier returned
two PID presentations under `pididentity01`; each had a valid `cnf.jwk`, and
the existing cryptographic checker also passed both KB-JWT signatures and
RFC 9901 `sd_hash` values. The reusable pipeline currently stores its visual
evidence as the holder-binding variant screenshot collection; 008 requires it
to be non-empty.

## Case 118

118 uses `claim_path_allowed_components` over one PID credential query with
three intended resolvable paths: `["given_name"]`,
`["nationality", null]`, and `["nationality", 0]`. The validator requires
non-empty path arrays, allows only strings, nulls, and non-negative integers,
requires evidence of all three categories, parses every returned SD-JWT, and
resolves every requested path against its disclosed claims. Unit evidence
rejects empty arrays, booleans, negative integers, fractional numbers, missing
component categories, missing presentations, and presentations that omit one
of the requested paths.

The public verifier accepted the combined request shape. A dedicated
`fcaf-test` Keycloak user was issued a fresh PID after its realm profile was
verified with `nationality: ["IT"]`, birth date, and structured birthplace.
On `emulator-5554`, the reference Wallet offered both the old Filippo PID and
the new FCAF PID. Expanding both consent rows showed only `Given Name(s)`;
`Nationality` was absent. Sharing completed and the verifier returned HTTP 200
with two presentations, but their only disclosures were respectively
`[salt, "given_name", "Filippo"]` and `[salt, "given_name", "FCAF"]`.
The strict Maestro flow therefore fails at the pre-share `Nationality`
assertion, and the protocol validator fails because neither nationality path
resolves. Keep both assertions strict; the observed Wallet behavior does not
satisfy case 118.

## Case 117

117 uses two distinct credential query IDs and omits `credential_sets`. The
existing `without_credential_sets` validator checks that the property is absent
and that the Wallet returns a non-empty presentation for every query ID; focused
unit cases prove that omitting either response entry fails. Maestro requires two
requested-document entries in one consent screen, expands both before sharing,
and expands both result entries after one Share/PIN interaction.

The request uses two distinct query IDs with the same PID `given_name` claim
constraint, and both may be satisfied by the same stored PID. Whether the upstream scenario instead
requires two distinct stored Credentials is ambiguous and is tracked in
`TEST-AUTHOR-FEEDBACK.md` Issue 16.

The reference-wallet Maestro run passed end to end on `emulator-5554`. The
post-link PIN must be entered digit by digit with zero settle time on the sixth
digit; otherwise Maestro waits through the short-lived request-screen
transition and observes Home. The Wallet displayed two PID request rows. One
tap on the first document expands both rows because their accordion state is
shared; both exposed `Given Name(s)` with value `Filippo`. After one Share/PIN
interaction, the success screen contained two document rows, both expanded by
one tap and both exposing `Given Name(s)`. The verifier returned HTTP 200 with
non-empty `pid-given-name` and `pid-given-name-copy` entries in `vp_token`.

The Wallet contains multiple matching PID instances and returned multiple
presentations under each query ID. Case 117 establishes the missing
`credential_sets` all-query requirement; cardinality when `multiple` is omitted
remains covered separately by case 071.

## Case 116

116 uses `claims_without_values` to require every claim to omit `values`, retain a valid non-empty string path, and produce a matching `vp_token` under the credential query ID. A parsed request or absence of an error cannot pass. The public verifier accepted the request shape and issued a request URI; the Maestro flow requires consent, PIN confirmation, and visible successful sharing.

The exact reusable Maestro flow failed because the reference Wallet did not show `DATA SHARING REQUEST`; the verifier transaction then returned HTTP 400 with an empty body. This is a failed case 116 result, not acceptance. Keep the strict presentation and successful-sharing assertions. The missing transaction diagnostics are tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 14.

## Case 115

115 uses `claim_path_non_array` with unit evidence for `null`, booleans, zero, a non-zero number, string, and object values. Missing paths, empty arrays, and valid non-empty arrays are explicitly excluded from this mode. Passing evidence requires no `vp_token` and a real `error: invalid_request`.

The public verifier rejected the representative string request with HTTP 400 because `ClaimPath` decoding expected an array, before the request could reach the Wallet. Device-level execution requires the raw mock-verifier service tracked in `TEST-AUTHOR-FEEDBACK.md` Issue 13.

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

The dedicated scenario sends two claims in the same credential query with the same non-empty `duplicated_name` ID. Capture Wallet accepts the request and preserves both IDs in its signed Authorization Request. The `duplicate_claim_ids` validator proves the duplicate, requires no `vp_token`, and requires a captured `invalid_request`; a duplicated ID across separate credential queries is explicitly not treated as this malformed case. An emulator run remains needed to establish the reference Wallet's conformance result.

## Case 109

The dedicated Capture Wallet scenario sends a credential query whose claim omits `id` and whose credential query omits `claim_sets`. Capture Wallet accepts and preserves that shape in its signed Authorization Request. The `claims_without_id_without_claim_sets` validator proves both omissions, validates the claim path, and requires a `vp_token` entry for the credential query ID; visual evidence is separately required. An emulator run remains needed to establish the reference Wallet's conformance result.

## Case 105

105 now has a dedicated `claims_present` validator requiring every credential query to contain a non-empty `claims` array, valid non-empty string paths, and a matching `vp_token`. The verifier accepted a fresh claims-bearing request and Maestro drove the wallet through PIN entry, but the wallet returned Home without showing the consent/share screen. Restarting the wallet process and retrying produced the same result.

TODO: finish 105 emulator diagnosis with runner-accessible wallet logcat and verifier transaction evidence. Direct ADB logcat is currently blocked because the sandbox cannot start the ADB smartsocket daemon; Maestro MCP can still inspect and drive the emulator. The verifier transaction endpoint returned HTTP 400 with an empty body after the wallet interaction.

## Case 106

106 uses a dedicated PID claim path, `claim_that_does_not_exist`, and the `claims_path_no_match` validator proves that the request contains a non-empty claim path and returns no `vp_token`.

Emulator evidence is incomplete: the verifier accepted the request, the Wallet accepted PIN `123456`, and then returned Home without consent or presentation. The verifier transaction endpoint returned HTTP 400 with an empty body. The source expects an observable `access_denied` response describing that no credentials match; current evidence proves only absence of a credential response. Keep this residual gap explicit until verifier diagnostics or protocol evidence are available.

## Case 107

107 uses the valid `given_name` path with a deliberately mismatched `values` constraint. The dedicated `claims_values_no_match` validator requires non-empty `path` and `values` arrays and proves that no `vp_token` is returned. A separate strict assertion requires the source-mandated `access_denied`; a silent discontinuation does not pass.

The emulator accepted the request and PIN, then returned Home without consent or presentation. The verifier transaction endpoint returned HTTP 400 with an empty body. As with 106, this proves no credential was returned but does not prove the source-required `access_denied` response or description.

## Case 108

The dedicated Capture Wallet scenario sends a credential query whose claim omits `id` while credential-level `claim_sets` is present. Capture Wallet accepts and preserves the malformed shape in its signed Authorization Request. The `claim_id_missing_with_claim_sets` validator proves the shape, rejects any presentation, and requires a captured `invalid_request`; visual evidence is separately required. An emulator run remains needed to establish the reference Wallet's conformance result.

## Case 100

The dedicated Capture Wallet scenario sends a `credential_sets.options` entry that references an unknown credential query ID. Capture Wallet accepts and preserves that malformed shape in its signed Authorization Request. The `credential_sets_options_invalid_references` validator proves the invalid reference, rejects any presentation, and requires a privacy-preserving error. An emulator run remains needed to establish the reference Wallet's conformance result.

## Case 098

The dedicated Capture Wallet scenario sends `credential_sets.options` as a string rather than an array. Capture Wallet accepts and preserves the malformed shape in its signed Authorization Request. The `credential_sets_options_non_array` validator proves the type error, rejects any presentation, and requires a captured `invalid_request`. An emulator run remains needed to establish the reference Wallet's conformance result.

## Case 097

The dedicated Capture Wallet scenario sends an empty `credential_sets.options` array. Capture Wallet accepts and preserves the malformed shape in its signed Authorization Request. The `credential_sets_options_empty` validator proves the empty array, rejects any presentation, and requires a captured `invalid_request`. An emulator run remains needed to establish the reference Wallet's conformance result.

## Case 096

The dedicated Capture Wallet scenario omits `credential_sets.options`. Capture Wallet accepts and preserves the malformed shape in its signed Authorization Request. The `credential_sets_options_missing` validator proves the omission, rejects any presentation, and requires a captured `invalid_request`. An emulator run remains needed to establish the reference Wallet's conformance result.

## CryptographicHash 001

The existing Capture Wallet DCQL scenario obtains a successful PID SD-JWT VP and now exposes its raw `vp_token` as `pid_sdjwt`. The `sdjwt.disclosure_digests_sha_256` validator requires disclosed claims, accepts the explicit or RFC default SHA-256 algorithm, and relies on the SD-JWT parser to recompute every disclosure digest and verify its `_sd` reference. An emulator run remains needed to establish the reference Wallet's conformance result.

## TextualEncoding 001

The existing Capture Wallet Encoding scenario sends a DCQL string path, `["given_name"]`, for the standard PID SD-JWT VC. The `claims_subset` assertion verifies that the string path resolves to a disclosed value and that the known unrequested sibling `family_name` is absent, proving selective claim resolution without requiring the source's unavailable Arthur Dent fixture. An emulator run remains needed to establish the reference Wallet's conformance result.

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

## Wallet step editor compatibility (2026-09-01)

The pipeline editor could not display inline `action_code` mobile-automation
steps ("Invalid pipeline step data"): `walletActionStepConfig.deserialize`
requires `action_id`. The 11 happy-flow scenarios that used inline code now
reference two stored wallet actions synced from
`config_templates/fcaf/imports/forkbomb-bv-andrea/wallet/`:
`fcaf-exercise-wallet-generic` (generic capture-session exercise, `deeplink`
parameter, fixed `exercise-wallet` screenshot label) and
`fcaf-submit-request-object-by-value` (`CLIENT_ID`/`REQUEST_OBJECT`
parameters). Both keep `version_id: installed_from_external_source`, which the
editor resolves as the external version. The editor now also round-trips
arbitrary step `parameters` on edit instead of only `deeplink`.

The ~90 fragmented one-test DCQL scenarios in the complete-validation
aggregate still use inline `action_code` and remain flagged in the editor;
converting them needs their distinct mock-deeplink flow templates extracted
first (see scenario sources under `scenarios/fcaf-wallet-solution-relying-party-dcql-*`).

## Response transport evidence definitions

`WS_RP_MS_ProtocolMessages__047`, `129`–`131`, `133`, `134`, and
`WS_RP_IA_MainInteraction__049`, `052`, `054` now use raw Capture session
evidence. The definitions remain verifier-blocked in the implementation
inventory pending a live reference-Wallet run: this environment has no
`INSTANCE`, `API_KEY`, or `FCAF_DEVICE_ID`, so no session or JWE evidence was
observed. Do not substitute screenshots or decrypted results. The next live
run must inspect `raw.request_uri_http` for 047 and
`raw.presentation_response_http.body` for all direct_post.jwt cases; 129–131
must validate the original compact JWE against delivered client metadata.

## Case 038

`WS_RP_MS_ProtocolMessages__038` uses the dedicated Capture Wallet plain
redirect-URI scenario. It creates an unsigned `request_delivery: plain`
Authorization Request with `client_id_scheme: redirect_uri`, preserves the
returned Authorization Request, and captures the session response and Maestro
screenshots. Assertions require the `redirect_uri:` client identifier prefix, a
returned `vp_token`, and non-empty visual evidence. The source and Capture
Wallet contract establish all three evidence boundaries; a live reference
Wallet run remains required before claiming a conformance result.

## Case 111

`WS_RP_MS_Metadata__111` now owns a dedicated Capture Wallet scenario using
`client_id_scheme: x509_hash` with signed, by-reference delivery. It binds the
exact session Request Object to JWS-signature and leaf-certificate hash
assertions, and requires visual evidence that the Wallet proceeds. A direct
Capture probe created the session and retrieved a compact signed Request Object
with `x509_hash:` client ID and `x5c` leaf certificate. An emulator is not
connected, so a live reference-Wallet run remains required before claiming a
conformance result.

## Case 129

`WS_RP_MS_Metadata__129` now owns a distinct Capture Wallet scenario that
uses Capture's default `x509_hash` client identifier. The validator recomputes
the base64url SHA-256 hash of the Request Object's `x5c` leaf and requires the
Wallet-flow screenshot evidence. A direct probe created the default session
and confirmed the returned client ID matches the leaf hash. An emulator is not
connected, so a live reference-Wallet run remains required before claiming a
conformance result.

## Case 131

`WS_RP_MS_Metadata__131` shares the exact default-`x509_hash` scenario owned
by case 129: the same Request Object must carry the leaf certificate whose
public key verifies its JWS signature. Its dedicated assertion uses
`jose.jws_signed_request`, which parses the `x5c` leaf and verifies the compact
JWS with that certificate's public key; visual evidence proves the Wallet flow
proceeded. An emulator is not connected, so a live reference-Wallet run remains
required before claiming a conformance result.

## Case 015

`WS_RP_IA_Metadata__015` owns the existing decentralized-identifier Capture
scenario. It explicitly requests `client_id_scheme: decentralized_identifier`,
serves the corresponding `did.json`, and supplies `vp_formats_supported` only
through `client_metadata`. The assertions verify the Request Object against the
DID-published key, verify that verifier metadata is exclusive to
`client_metadata`, and require Wallet-flow visual evidence. A direct Capture
probe returned the decentralized client ID, signed Request Object, resolvable
DID document, and expected client metadata. An emulator is not connected, so a
live reference-Wallet run remains required before claiming a conformance result.

## Case 016

`WS_RP_IA_Metadata__016` owns the empty-client-metadata decentralized-
identifier Capture scenario. It captures the signed Request Object, its DID
document, the complete session response, and Wallet-flow screenshots. The
assertions verify the DID key before requiring `client_metadata: {}` and an
`invalid_request` response, preventing a signature failure from being treated
as the required metadata rejection. A direct Capture probe returned the
decentralized client ID, DID-key `kid`, and exactly empty client metadata. An
emulator is not connected, so a live reference-Wallet run remains required
before claiming a conformance result.

## Case 047

`WS_RP_MS_ProtocolMessages__047` now keeps the Wallet session free of manual
request-URI retrievals: its session capture proves the Wallet's `POST` and
Host header, while a separate same-shape Capture probe proves the endpoint's
`application/oauth-authz-req+jwt` response and compact signed Request Object.
The separate probe avoids falsely attributing an agent-issued `POST` to the
Wallet. A direct Capture probe confirmed the media type and a three-part JWS.
An emulator is not connected, so a live reference-Wallet run remains required
before claiming a conformance result.

## Case 129 (Protocol messages)

`WS_RP_MS_ProtocolMessages__129` uses the direct-post.jwt response-transport
scenario. It requires captured `POST` form evidence with exactly a `response`
parameter, then parses the original compact JWE's protected header and requires
its `kid` to occur in the Request Object's client-metadata JWK set. Screenshot
evidence proves the Wallet flow proceeded. An emulator is not connected, so a
live reference-Wallet run remains required before claiming a conformance result.

## Case 130

`WS_RP_MS_ProtocolMessages__130` owns the replacement-metadata response-
encryption scenario. The replacement retains Capture's generated JWK while
setting `authorization_encrypted_response_enc: A256GCM`; assertions require a
captured direct-post.jwt response, that exact metadata value and protected JWE
`enc`, and byte-equivalent preservation of the generated JWK set. An emulator
is not connected, so a live reference-Wallet run remains required before
claiming a conformance result.

## Case 131 (Protocol messages)

`WS_RP_MS_ProtocolMessages__131` shares the default direct-post.jwt scenario.
It requires a captured `POST` response containing only `response`, confirms
that signed request metadata omits the encryption override, and checks the
original compact JWE protected header defaults `enc` to `A128GCM`. An emulator
is not connected, so a live reference-Wallet run remains required before
claiming a conformance result.

## Case 133

`WS_RP_MS_ProtocolMessages__133` shares the default direct-post.jwt scenario
and requires raw Capture HTTP evidence: the response must use `POST` and contain
only a non-empty `response` form parameter. Screenshot evidence proves the
Wallet flow proceeded. An emulator is not connected, so a live reference-Wallet
run remains required before claiming a conformance result.

## Case 134

`WS_RP_MS_ProtocolMessages__134` shares the default direct-post.jwt scenario.
It requires raw HTTP `POST`, `application/x-www-form-urlencoded`, valid UTF-8,
and a body containing only `response`; the value must be an original compact
JWE with default `A128GCM` encryption. An emulator is not connected, so a live
reference-Wallet run remains required before claiming a conformance result.

## Case 049

`WS_RP_IA_MainInteraction__049` shares the default direct-post.jwt scenario and
requires the captured Wallet exchange to be a `POST` carrying
`application/x-www-form-urlencoded` with exactly one non-empty `response`
parameter. `WS_RP_IA_MainInteraction__052` additionally requires every body
name and value to be valid UTF-8, and `WS_RP_IA_MainInteraction__054` requires
the Authorization Response to be delivered by `POST` at all. An emulator is not
connected, so a live reference-Wallet run remains required before claiming a
conformance result.

## Response-transport assertion scope

OID4VP 1.0 section 8.3.1 says the Wallet "adds the `response` parameter
containing the JWT" to a UTF-8 `application/x-www-form-urlencoded` POST body,
and section 8.3 puts the section 8.1 response parameters inside the JWE
payload. The spec never says the body carries `response` *exclusively*, and
section 13.3 shows `state` as a separate direct-post response parameter.
Therefore `response_only: true` stays only on
`WS_RP_MS_ProtocolMessages__134`, whose source test explicitly demands a body
containing only the response parameter; the sibling rows
`WS_RP_MS_ProtocolMessages__129`/`130`/`131`/`133` and
`WS_RP_IA_MainInteraction__049`/`052`/`054` keep
`require_response_parameter: true`, which removes the vacuous pass on an empty
body without importing another row's criterion. `WS_RP_MS_ProtocolMessages__134`
is the only row that carried `response_only` before this work; do not copy it
to rows whose own source test does not require exclusivity.

## Case 057

`WS_RP_IA_MainInteraction__057` now owns a dedicated scenario,
`scenarios/fcaf-wallet-solution-relying-party-callback-redirect.yaml`, that
creates a `direct_post.jwt` session with a configured `redirect_uri`. The
scenario exposes the verifier's reply to the Wallet
(`raw.presentation_response_verifier_http`) together with the `redirect_uri`
echoed by session creation, and the new `oid4vp.response_endpoint_callback`
validator requires HTTP 200, `application/json`, `Cache-Control: no-store`, and
a callback `redirect_uri` targeting the configured URI.

Capture appends its own fresh `response_code` to the configured redirect URI, so
the validator compares scheme, host, and path only. Direct probes confirmed the
verifier-callback record shape, the `no-store` JSON error response, and the
session-creation `redirect_uri` echo; the successful callback body itself still
requires a live reference-Wallet run.

`WS_RP_IA_MainInteraction__061` and `067` share that scenario: 061 requires the
JSON redirect response the Wallet must open unchanged, and 067 requires the
callback to supply the configured redirect URI that the Wallet follows.

## Redirect-visit capture rework, 16/09/2026

The earlier limit recorded here - "Capture records no evidence of the Wallet's
navigation: `GET /openid4vp/redirect` returns 404, and opening a session's
configured redirect creates no session event or raw record" - is SUPERSEDED. It
was true only because those scenarios configured `redirect_uri:
https://verifier.eudiw.dev/`, an external URL, and because a bare
`/openid4vp/redirect` without a valid `response_code` still 404s.

Beta hosts a redirect capture page at
`https://beta-capture-wallet.credimi.io/openid4vp/redirect`. Probed live with
both that concrete URL and the upstream `{{base_url}}/openid4vp/redirect`
template: session creation returns the URI with a fresh `response_code`, and a
`GET` of that exact URI returns the
`Presentation complete` page and records `redirect_uri_visited_at`,
`redirect_uri_visit_count`, a `vp_redirect_uri_visited` event, and a
`raw.redirect_uri_visits` envelope. An unknown, empty, or missing
`response_code` returns 404 and records nothing, so a recorded visit identifies
the exact URI that was opened. Guardrails and the full probe are in
`CAPTURE_WALLET_API.md`.

Consequently:

- `scenarios/fcaf-wallet-solution-relying-party-callback-redirect.yaml` and
  `scenarios/fcaf-wallet-solution-relying-party-supportive-redirect-uri.yaml`
  now configure the concrete beta redirect page through
  `${fixture.verifier_url}/openid4vp/redirect` and expose `capture_session`.
  Scenarios must not use the `{{base_url}}` template: the generated pipeline has
  to show the real target rather than rely on deployment-side resolution. The
  supportive Maestro flow waits for `Presentation complete` instead of the
  `verifier.eudiw.dev` landing page, and its `openLink` indentation, which made
  the flow unparseable, is fixed.
- New validator `oid4vp.redirect_uri_visited` requires the visit counter, the
  timestamp, the retained envelope, and the capture event to agree, with
  `min_visits` and `after_presentation_response` for ordering against
  `vp_presentation_response_received`. Verified against two live sessions: a
  response-then-visit session passes the ordering check, a visit-only session
  fails it and passes without it.
- `057`, `061`, `067` each gained that assertion, so the navigation half no
  longer rests on `evidence.non_empty` screenshots. `061` additionally requires
  `require_response_code` on the callback body.
- `WS_RP_IA_Supportive__001` asserted only that a `vp_token` was posted, while
  its subject is following the redirect. It now asserts the visit too.
- `WS_RP_IA_MainInteraction__054`'s assertion id claimed the response URI was
  proven; capture records no request target, so it is now
  `authorization_response_is_delivered_by_post`.

Remaining honest limits:

- `raw.redirect_uri_visits` records method and redacted headers only, with no
  request target or query, so the "does not append the Authorization Response to
  the redirect_uri" half of 061 still rests on screenshots.
- The callback body's `response_code` is required to be present, not equal to
  the session-creation value: OID4VP 8.2 lets the Verifier generate the code
  when it receives the response, and no live Wallet submission has been observed
  to settle whether Capture reuses the creation-time code.
- Any client can create a visit. No pipeline step may fetch a session's
  configured redirect URI, or the evidence is fabricated.
- 061 still does not assert `Cache-Control: no-store`. The service always sends
  it, so the check cannot fail and would only pad the assertion set; the suite
  certifies the Wallet, not the verifier's caching.

## Cases 137, 138, 139, 140, 142, 150

These six request-shape/error cases now use two new validators.
`oid4vp.error_response_required` requires the Wallet's exact error code and no
`vp_token`, which is what "does not proceed to credential selection" means;
`oid4vp.session_event_count` requires an exact capture event count. The existing
`dcql.response_satisfies_constraints` could not serve them: it hard-fails unless
the evidence contains a `dcql_query`, and `dcql_query: null` sessions contain
none anywhere.

Four defects were found and fixed while reviewing the sources:

- Capture substitutes its default DCQL query when `presentation_request.dcql_query`
  is omitted. `142` omitted the key and therefore delivered a fully valid request
  while claiming "no data requirements". It now sends `dcql_query: null`, which
  verifiably removes the query from the signed request.
- `138` sent the same empty scope as `139`. It now sends `bad\scope`, a backslash
  being illegal in an RFC 6749 scope-token.
- `142` asserted `invalid_scope`; its source requires `invalid_request`.
- `137`/`138`/`139`/`140` sent a valid `dcql_query` alongside the scope, which
  risks the ambiguous "both dcql_query and a scope" case from OID4VP 8.5. They
  now send `dcql_query: null` plus the scope only.

`scope` placement matters: it must sit inside `presentation_request` to reach the
signed request. A top-level `scopes` field is accepted but ignored, so it never
reaches the Wallet. Verified by decoding the signed Request Object.

`140` genuinely proves termination: every Wallet delivery records a
`vp_presentation_response_received` event, so exactly one such event means the
Wallet sent its error and then stopped. A second delivery adds a second event and
fails the test.

`150` still asks for `format: vc+sd-jwt`, which Capture accepts and preserves. The
source precondition says the Wallet supports `mso_mdoc` but not `vc+sd-jwt`; the
reference Wallet supports the current `dc+sd-jwt` name, so whether it treats the
pre-final alias as unsupported is a live-run question, not a definition defect.

An emulator is not connected, so none of these six has a live reference-Wallet
result yet.

## Case 006, RpIntegrity decentralized identifier

`WS_RP_SM_RpIntegrity__006` now uses the DID metadata scenario rather than the
generic RP-integrity scenario. It fetches the compact signed Request Object from
the Capture `request_uri` and the verifier DID document, then verifies that the
`kid` selects a P-256 DID key that validates the JWS. The validator also
requires `client_id` to begin `decentralized_identifier:did:`. The test requires
one `vp_presentation_response_received` event and visual evidence.

A live beta probe confirmed that a decentralized-identifier request exposes the
signed Request Object through `request_uri`; the session JSON only exposes a
decoded Authorization Request. No mobile runner was available to execute the
Wallet interaction.

## Case 013, RpIntegrity X.509

`WS_RP_SM_RpIntegrity__013` uses the `x509_hash` signed-request scenario. It
retrieves the compact JWS from the Capture `request_uri`, verifies the JWS with
the leaf key in its `x5c` chain, and checks the `x509_hash` client ID against
that same leaf certificate. The case also requires a single presentation
response event and visual evidence. A live beta probe confirmed an ES256
`oauth-authz-req+jwt` with one `x5c` certificate and an `x509_hash` client ID.
No mobile runner was available for the Wallet interaction.

## Cases 016, 018, 020, and 023, RpIntegrity X.509

The Capture `x5c` header deliberately contains the signing leaf only. Its AIA
issuer, `PID Issuer CA 02`, is a self-issued CA certificate, so the missing
root is expected: OpenID4VP trust anchors belong in the Wallet trust store, not
in `x5c`. Cases 016, 018, and 023 verify the leaf-key JWS signature, the
`x509_hash` client-identifier binding, and the Wallet presentation-response
event; 023 covers acceptance of the `x509_hash` Client Identifier Prefix, where
the response event plus screenshots evidence consent and a successful
presentation. Case 020 additionally requires `vp_formats_supported` exclusively
in `client_metadata`.

## Case 028, RpIntegrity unsigned request

`WS_RP_SM_RpIntegrity__028` uses the plain `redirect_uri` scenario, which is
the only source that delivers request parameters without a Request Object. A
live beta Capture probe of `request_delivery: plain` returned an
`authorization_request` with neither `request` nor `request_uri`, and the
deeplink carried the parameters inline, so both absence assertions fail if the
verifier ever switches that source back to signed or by-reference delivery.
Successful handling is proven by exactly one `vp_presentation_response_received`
capture event. The session body keeps a bookkeeping top-level `request_uri`
even for plain delivery, so absence must be asserted on `authorization_request`
only.

## Case 029, RpIntegrity signed request

`WS_RP_SM_RpIntegrity__029` uses the default `x509_hash` scenario, which signs
with the verifier default rather than an explicitly requested prefix, keeping it
distinct from case 023. The scenario now also exports `capture_session`, so the
Wallet-answers half is proven by one `vp_presentation_response_received` event
instead of screenshots alone. A live beta probe of `request_delivery:
by_reference` returned an ES256 `oauth-authz-req+jwt` at
`raw.authorization_request_jwt`; engine runs confirmed the definition passes on
that JWS, fails when the signature is tampered, and fails when no response
event was captured.

## Case 031, RpIntegrity ES256 signed request

`WS_RP_SM_RpIntegrity__031` reuses the case 029 evidence and adds the HAIP §7
algorithm requirement: `jwt.header_field_equals` pins header `alg` to `ES256`
before `jose.jws_signed_request` verifies the signature, so a P-384 or RS256
verifier switch fails loudly instead of passing on a valid-but-wrong signature.
A live beta probe confirmed the default Capture header is
`{alg: ES256, typ: oauth-authz-req+jwt}`. Engine runs confirmed the definition
passes on that JWS, fails when the header is rewritten to `ES384`, and fails
when no response event was captured.

## Case 001, SessionEncryption unsigned encrypted response

`WS_RP_SM_SessionEncryption__001` keeps the session-encryption scenario and now
asserts the source requirement instead of a valueless `alg` header check: the
session ran with `response_mode: direct_post.jwt`, the Wallet response is a
compact JWE, its protected header carries `enc`, and it carries no `cty`. The
`cty` check is the RFC 7519 section 5.2 signal for nested signing, which is the
only observable way to reject a signed-then-encrypted response without the
verifier decryption key. Do not assert `typ` absence: RFC 7519 allows `typ:
JWT` on a compliant encrypted JWT.

A live beta probe confirmed the session exposes top-level `direct_post.jwt` and
publishes one `use: enc` ECDH-ES P-256 JWK; `raw.presentation_response` only
appears once a Wallet has answered, so engine runs used a wallet-shaped compact
JWE built against that published JWK. The validators never decrypt, so the
structural fixture exercises the same code path a live response would.

## Cases 002 and 003, SessionEncryption verifier encryption JWK

Both require a `direct_post.jwt` request whose `client_metadata` publishes a
deliberately unusable verifier encryption JWK, after which the Wallet must
return an error instead of a presentation. OID4VP section 8.3 states "The alg
parameter MUST be present in the JWKs" and "The JWE alg algorithm used MUST be
equal to the alg value of the chosen jwk", so each request is malformed and
`invalid_request` is the expected code, matching
`WS_RP_MS_ProtocolMessages__142`.

The capability that unblocked them is `allow_undecryptable_response: true`,
which waives the verifier-encryption-key check and publishes the supplied
`jwks` verbatim. Each case owns a scenario with a static, on-curve P-256
`use: enc` key:

- `002`: `fcaf-wallet-solution-relying-party-response-encryption-jwk-without-alg`
  omits `alg` entirely.
- `003`: `fcaf-wallet-solution-relying-party-response-encryption-jwk-alg-mismatch`
  sets `alg: ECDH-ES+A256KW`. HAIP pins the JWE `alg` to bare `ECDH-ES`, so the
  published value cannot be the `alg` the Wallet would use, and it must refuse
  rather than silently substituting `ECDH-ES`.

Observed on beta 17/09/2026: both bodies returned `201`, published the key
exactly as supplied, and recorded a `vp_undecryptable_response_allowed` event.
Use a genuine curve point: an earlier placeholder was off-curve, which would
have let a Wallet refuse for the wrong reason.

`oid4vp.request_encryption_jwk` asserts the published key: `{field: alg,
present: false}` for `002` and `{field: alg, value: ECDH-ES+A256KW}` for `003`.
It selects the first `use: enc` key, falling back to the only published key.
The refusal half reuses `oid4vp.session_event_count` with `count: 0` plus
`oid4vp.error_response_required`. Engine runs confirmed each case passes only on
its own scenario evidence, fails on the other's, and fails when the session
recorded a presentation response.

The captured shape of a real Wallet error is still unverified: no mobile runner
was available, so the refusal fixtures used the established
`oid4vp.error_response_required` contract rather than an observed error capture.

Related finding, not fixed here:
`fcaf-wallet-solution-relying-party-response-encryption-metadata.yaml` still
replaces `jwks` at the body top level without the new flag, so it returns HTTP
400. `WS_RP_MS_ProtocolMessages__130` depends on that scenario.

## Case 005, SessionEncryption ECDH-ES on P-256

`WS_RP_SM_SessionEncryption__005` asserts the RFC 7516 section 4.1.1 header
`alg` is exactly `ECDH-ES` and the RFC 7518 section 6.2.1.1 ephemeral key uses
`epk.crv: P-256`, plus the `direct_post.jwt` response mode and exactly one
`vp_presentation_response_received` event so the encrypted response was really
delivered. Pinning `alg` to bare `ECDH-ES` rejects the wrapped variants such as
`ECDH-ES+A128KW`, which the source does not allow.

A live beta probe confirmed the verifier publishes an `ECDH-ES` P-256 `use: enc`
JWK for this scenario. Engine runs confirmed the definition passes on a
wallet-shaped `ECDH-ES` P-256 JWE, and that each half fails alone:
`ECDH-ES+A128KW` fails only the algorithm assertion, a P-384 `epk` fails only
the curve assertion, and a session without the response event fails only the
delivery assertion.

## Case 012, SessionEncryption redirect-flow encrypted response

`WS_RP_SM_SessionEncryption__012` moved from the session-encryption scenario to
`pipeline.direct-post-jwt.response-transport`, because only that source exports
`default_encryption_evidence` (`request_object` plus
`presentation_response_http`), which is what the two distinguishing halves of
the case need: the response is encrypted "using parameters provided in
client_metadata" (`oid4vp.response_encryption` with `match_metadata_kid`), and
the Verifier "is able to decrypt" it (`dcql.response_satisfies_constraints` in
`credentials_match` mode, which only passes when the session exposes a
decrypted `vp_token` keyed by the credential query ID).

A live beta probe confirmed the `request_uri` POST returns a Request Object
whose `client_metadata.jwks` kid equals the kid in the session metadata, so the
kid match is a real cross-check rather than a tautology. Engine runs confirmed
the definition passes on a metadata-keyed response, that a foreign `kid` fails
only the client-metadata assertion, and that a session without a decrypted
`vp_token` fails only the decryption assertion.

## Case 022, TextualEncoding direct-post body UTF-8

`WS_RP_SH_Encoding_TextualEncoding_022` requires a `direct_post.jwt` POST whose
body names and values are UTF-8, so it moved from the `direct_post` encoding
scenario to `pipeline.direct-post-jwt.response-transport`, the only source with
both the required response mode and `raw.presentation_response_http`. The old
`sdjwt.claim_utf8_string` assertion tested credential claims, not the POST body
the source asks about.

`form_utf8` is a real discriminator even though a `direct_post.jwt` body is
base64url ASCII: `oid4vp.presentation_response_http` percent-decodes the form
first, so a Latin-1 escape such as `state=Ana%20Mar%EDa` fails while the UTF-8
form `%C3%AD` passes. Engine runs confirmed both, plus a JSON body failing the
form media type. The source precondition asks for a non-ASCII credential, but
with an encrypted response the credential bytes never reach the body, so the
observable requirement is the form encoding itself.

## Cases 007, 008, and 009, SessionEncryption content encryption

All three check the RFC 7516 section 4.1.2 `enc` header of the Wallet response,
so they use `pipeline.direct-post-jwt.response-transport`, whose
`default_encryption_evidence` carries both the request object (for the
advertised algorithms) and the raw response form (for the chosen algorithm).
Each case must pair its own advertisement with the selection, because the
profile differs per case:

- `007`: only A128GCM offered, response `enc` is A128GCM.
- `008`: only A256GCM offered, response `enc` is A256GCM.
- `009`: both offered, response `enc` must be A256GCM.

`oid4vp.response_encryption` gained two parameters for this:
`metadata_enc_values_supported` requires
`client_metadata.encrypted_response_enc_values_supported` to contain each listed
value, and `metadata_enc_values_exclusive` requires the advertised set to hold
nothing else. The pre-existing `metadata_enc` parameter only reads the old draft
scalar `authorization_encrypted_response_enc`, which beta Capture no longer
emits, so it could not express any of these cases.

Each case now owns a scenario that advertises exactly what its profile
describes, because the previous shared source made the advertisement half
vacuous: Capture's generated `[A128GCM, A256GCM, A128CBC-HS256]` contains every
value, so containment passed for all three and each case degraded to a bare
`enc` equality check.

- `007`: `fcaf-wallet-solution-relying-party-response-encryption-a128gcm`
- `008`: `fcaf-wallet-solution-relying-party-response-encryption-a256gcm`
- `009`: `fcaf-wallet-solution-relying-party-response-encryption-both-gcm`

They rely on the one-level-deep `client_metadata` merge: supplying only
`encrypted_response_enc_values_supported` narrows the advertisement while the
generated `jwks` and `vp_formats_supported` survive, so the service still
decrypts. Observed on beta 17/09/2026: `[A128GCM]`, `[A256GCM]`, and
`[A128GCM, A256GCM]` were each emitted verbatim with the generated key intact,
and every delivered Request Object carried the intended list. The production
deployment still returned the full three-value list on that date, so these
scenarios run on beta only.

All three use `metadata_enc_values_exclusive: true`, so a source whose
advertisement drifts fails loudly instead of passing on containment. Engine runs
confirmed each case passes on its own scenario evidence with the matching
response `enc`, fails when the response uses the other GCM length, and fails
when given another scenario's request object.

## Case 011, SessionEncryption per-request ephemeral key

`WS_RP_SM_SessionEncryption__011` needs two Authorization Requests carrying
different verifier ephemeral encryption keys, and each response must be
encrypted to its own request's key. It owns
`fcaf-wallet-solution-relying-party-response-encryption-per-request-keys`, which
creates two `direct_post.jwt` sessions, exercises the Wallet once per deeplink,
and exports both request objects plus a per-session encryption evidence object.

Capture mints a fresh encryption key per session, so no metadata override is
needed: a probe on beta 17/09/2026 returned distinct `kid` and distinct key
material for two consecutive sessions.

The pairing is asserted with `oid4vp.response_encryption`
(`match_metadata_kid`) once per session. That is only meaningful while the two
published keys really differ, so the new
`oid4vp.distinct_request_encryption_keys` validator asserts distinctness first.
It compares the public key members (`kty`, `crv`, `x`, `y`, `n`, `e`) rather
than `kid`, so a reused key relabelled with a new `kid` is still caught, and
`minimum_keys` guards against a scenario that silently degrades to one request.

Engine runs confirmed: both responses keyed to their own request pass; a second
response reusing the first key fails only the second pairing assertion; and two
identical request objects fail the distinctness assertion, so a verifier that
stopped rotating keys cannot yield a false pass.
