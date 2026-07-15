<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# FCAF test-author feedback

Copy-paste-ready issue drafts collected while implementing the wallet-solution relying-party FCAF tests. Findings were verified against the local FCAF source checkout corresponding to links pinned at commit `88ab69a` in the implementation inventory.

## Upstream FCAF issues

### Issue 1: Use the normative `values` property in trusted-authorities tests 090-094

Suggested title:

```text
fix(wallet-rp): use trusted_authorities.values in tests 090-094
```

Suggested issue body:

```markdown
The wallet relying-party protocol-message tests 090-094 refer to a singular `value` property:

- `WS_RP_MS_ProtocolMessages__090`: missing `value`
- `WS_RP_MS_ProtocolMessages__092`: `value` is not an array
- `WS_RP_MS_ProtocolMessages__093`: `value` contains a non-string item
- `WS_RP_MS_ProtocolMessages__094`: `value` contains an empty string

OID4VP 1.0 Section 6.1.1 defines the required property as plural `values`: a non-empty array of strings. The singular word “value” is only used when describing one item inside that array.

This discrepancy makes it unclear whether an implementation should omit/mutate `value` or the normative `values` member. A literal implementation of the test prose would test an unrelated extension property.

Proposed change:

- Replace property references from `value` to `values` in objectives and scenarios for 090, 092, 093, and 094.
- In 090, explicitly state that the request contains a valid `type` and omits `values`.
- Link directly to OID4VP Section 6.1.1.

Normative reference: https://openid.net/specs/openid-4-verifiable-presentations-1_0.html#section-6.1.1
```

### Issue 2: Correct duplicate email-address test ID `_003`

Suggested title:

```text
fix(wallet-rp): correct duplicate email-address test identifier
```

Suggested issue body:

```markdown
Two different files currently declare the same H1 test ID:

- `DataModel/AddressData/WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_002.md`
  - H1: `WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003`
  - Objective: verify UTF-8 string encoding
- `DataModel/AddressData/WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003.md`
  - H1: `WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003`
  - Objective: verify RFC 5322 email-address syntax

The `_002.md` filename and its distinct objective strongly indicate that its H1 should end in `_002`.

Impact:

- Duplicate IDs break deterministic catalog generation.
- Links and implementation mappings can point to the wrong objective.
- Result reporting cannot distinguish the two tests.

Proposed change: change the H1 in `_002.md` to `WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_002` and add a CI uniqueness check for test headings.
```

### Issue 3: Remove the accidental `%` from `WS_RP_IA_Supportive_%001`

Suggested title:

```text
fix(wallet-rp): remove percent sign from supportive test 001 ID
```

Suggested issue body:

```markdown
`Interaction/Supportive/WS_RP_IA_Supportive__001.md` declares:

`# WS_RP_IA_Supportive_%001`

The percent sign is inconsistent with the filename, neighboring test IDs, and the general identifier vocabulary. It also requires URL encoding and can be interpreted as the start of percent-encoded data by downstream tooling.

Proposed canonical ID:

`WS_RP_IA_Supportive_001`

Please update the H1 and any generated indexes/references. A CI rule should reject `%` and other non-identifier punctuation in test IDs.
```

### Issue 4: Add automated filename, heading, and uniqueness validation

Suggested title:

```text
ci(fcaf): validate canonical test IDs across filenames and headings
```

Suggested issue body:

```markdown
The wallet relying-party source contains multiple identifier inconsistencies, including:

- duplicate H1 IDs in distinct files;
- `%` inside an ID;
- filenames using `__NNN` while headings commonly use `_NNN`;
- missing separators before the sequence in IDs such as `CredentialStructure010`, `CryptographicHash001`, and `TextualEncoding001`;
- content-family mismatches, for example `Familynamebirth` files whose H1 identifies `Familyname` tests and `Givennamebirth` files whose H1 identifies `Givenname` tests.

These inconsistencies make the filename, H1, and generated indexes disagree about the source-of-truth identifier.

Proposed CI checks:

1. Define one canonical ID grammar.
2. Normalize the filename stem and require it to equal the H1 ID.
3. Require every H1 test ID to be globally unique.
4. Reject unexpected punctuation such as `%`.
5. Report every mismatch with both paths before generation.

Please document whether the double underscore before numeric suffixes is intentional. If it is only a filename convention, generators should normalize it explicitly rather than infer ad hoc mappings.
```

### Issue 5: Correct `claims_sets` typo in protocol-message test 086

Suggested title:

```text
fix(wallet-rp): correct claim_sets typo in protocol test 086
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages__086.md` correctly uses `claim_sets` in its objective and scenario, but Expected result 3 says:

`The wallet detects an invalid credential "claims_sets" property...`

The normative DCQL property is `claim_sets`, not `claims_sets`.

Proposed change: replace `claims_sets` with `claim_sets` and format the property as code consistently throughout the test.
```

### Issue 6: Define observable criteria for “discontinuing the interaction”

Suggested title:

```text
docs(wallet-rp): make malformed-request rejection outcomes observable
```

Suggested issue body:

```markdown
Several malformed-request tests, including protocol-message tests 088-094, allow the Wallet to reject by:

1. returning `invalid_request` with details;
2. returning an unspecified error; or
3. discontinuing the interaction.

The third outcome is not operationally defined. During reference-wallet execution, a malformed request can simply return to Home without an error screen. It is difficult to distinguish this from a deep link that was never received, an app-routing failure, timeout, or test harness error.

Proposed additions to each affected test:

- Define evidence that proves the Wallet received the request before discontinuation.
- Define the observation window/timeout.
- State whether returning to Home is sufficient.
- Require proof that no consent screen and no successful authorization response were produced.
- If a verifier endpoint is used, require evidence that no `vp_token` was accepted.

This would make all three permitted outcomes objectively testable without assuming that silence equals rejection.
```

### Issue 7: Specify response mapping and UI behavior for same-credential queries in test 087

Suggested title:

```text
clarify(wallet-rp): define evidence for two queries matching one credential
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages__087` says two credential queries match the same wallet credential and that the Wallet must not return an error or force the user to pick the credential twice.

The expected result does not define the protocol response shape or the expected consent UI. Implementations need clarity on at least:

- whether the `vp_token` object must contain entries keyed by both credential-query IDs;
- whether both entries may contain presentations derived from the same stored credential;
- whether one consent screen may display two requested-document rows;
- what “pick the credential twice” means when no explicit credential picker is shown;
- whether one Share/PIN action is required for the complete request.

Observed reference-wallet behavior was one consent interaction containing two PID rows, followed by one Share/PIN action and a successful response containing both query IDs.

Proposed change: make the expected protocol mapping and consent interaction explicit, while allowing equivalent UI designs that do not require duplicate user selection.
```

### Issue 8: Clarify and deduplicate DCQL scope tests 019-021

Suggested title:

```text
clarify(wallet-rp): define supported DCQL scope values in tests 019-021
```

Suggested issue body:

```markdown
Protocol-message tests 019-021 contain unclear or duplicated wording:

- 019 says a Wallet processes either `dcql_query` or a scope representing a DCQL query.
- 020 repeats essentially the same objective but its scenario tests only a scope.
- 020 contains the typo `ascope`.
- 019-021 include the editorial note: “Need to check if EUDIW supports both by DCQL and scopes - and if scopes are supported which values are defined.”
- 021 tests both mechanisms simultaneously, overlapping later protocol-message test 141.

The tests do not name any scope value that maps to a known DCQL query, so an implementation cannot construct an interoperable request. Some wallet cores expose no known scope-to-DCQL mapping at all.

Proposed changes:

1. Decide whether scope-based DCQL queries are mandatory, optional, or implementation-specific.
2. Define the exact scope values and corresponding DCQL query for applicable profiles.
3. Give 019 and 020 distinct objectives.
4. Remove editorial notes from normative scenarios.
5. Resolve overlap between 021 and 141.
6. Fix `ascope` to `a scope`.
```

### Issue 9: Complete empty applicability and precondition sections

Suggested title:

```text
docs(fcaf): require explicit applicability and preconditions in test definitions
```

Suggested issue body:

```markdown
Many wallet relying-party tests contain empty `Profile applicability` and `Preconditions` sections even when the scenario depends on concrete wallet state, credential format, credential count, verifier behavior, or transport profile.

Examples:

- malformed `trusted_authorities` tests do not state the base credential format or matching credential requirement;
- test 087 does not state that exactly one stored credential must match both queries;
- test 071 says more than one credential must match but does not define how those credentials differ or how the verifier proves both were returned.

Empty sections force implementers to infer setup from neighboring tests and can produce incompatible test vectors.

Proposed change:

- Require each section to contain an explicit value or `None` with rationale.
- Define credential format, credential type, minimum stored credentials, trust prerequisites, and presentation transport where relevant.
- Add a documentation/schema check rejecting empty required sections.
```

### Issue 10: Make `multiple: true` evidence requirements explicit in test 071

Suggested title:

```text
clarify(wallet-rp): define protocol and consent evidence for multiple true
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages__071` requires the Wallet to process `multiple: TRUE` and return multiple credentials, but it does not define:

- the minimum number of matching credentials;
- how those credentials are made distinguishable in the test fixture;
- the exact `vp_token` shape and minimum presentation count;
- whether the user must be shown a multi-credential selection/consent UI;
- what visual evidence is required to prove the Wallet considered multiple credentials rather than duplicating one response.

Proposed expected evidence:

1. At least two distinct matching credentials exist before the request.
2. The query contains JSON boolean `multiple: true`.
3. The response entry for the query ID contains at least two presentations.
4. The presentations are shown to represent distinct stored credentials.
5. Consent/selection evidence is captured where the Wallet UI exposes it.
```

## Credimi inventory and tooling issues

### Issue 11: Prevent duplicate inventory rows when implementation status is overlaid

Suggested title:

```text
fix(fcaf): merge implementation status into canonical inventory rows
```

Suggested issue body:

```markdown
`pkg/fcaf/implementation-inventory.csv` can contain two rows for the same upstream test ID: an original `adapt manual` row and a later `ready` or `failing` implementation row. Cases 086 and 087 have exhibited this pattern.

Impact:

- Counts are inflated.
- Consumers cannot determine the authoritative status.
- Generated reports may show contradictory readiness for one test.

Proposed change:

- Key inventory generation by canonical upstream source path/test ID.
- Update the existing row when Credimi evidence is added instead of appending a second row.
- Fail generation when a canonical key occurs more than once.
- Add a regression test for duplicate source paths and normalized IDs.
```

### Issue 12: Separate request-shape evidence from observed wallet rejection

Suggested title:

```text
test(fcaf): stop synthesizing invalid_request evidence for UI discontinuation
```

Suggested issue body:

```markdown
Some Credimi malformed-DCQL pipelines construct `json-parse` evidence containing `error: invalid_request` even when the observed reference-wallet behavior is only interaction discontinuation (for example, returning to Home).

This proves the intended expectation, not the actual wallet response.

Proposed change:

- Store the malformed request shape separately.
- Capture actual verifier response/error data when available.
- Represent UI-only discontinuation explicitly rather than inserting `invalid_request`.
- Require both request-shape validation and observed rejection/discontinuation evidence.
- Keep `request_rejected` semantically distinct from credential `no_match`.
```

## Suggested upstream validation rules

A single source-quality job could prevent several findings above:

```text
- filename-derived ID matches H1 after one documented normalization rule
- H1 IDs are globally unique
- IDs match a declared grammar and contain no percent signs
- Profile applicability and Preconditions are non-empty or explicitly None
- referenced protocol property names belong to a checked vocabulary
- no editorial TODO/Need to check text remains in published test scenarios
- generated inventory contains one row per canonical source test
```

### Issue 13: Provide a mock verifier for malformed DCQL requests

Suggested title:

```text
test(fcaf): provide raw mock-verifier support for malformed DCQL cases
```

Suggested issue body:

```markdown
FCAF cases 096–098, 100, 108, 110–115, and 120–123 require the Wallet to receive DCQL that the
public verifier rejects before producing a signed request:

- 096: `options` is missing
- 097: `options` is an empty array
- 098: `options` is not an array
- 100: `options` references an unknown credential query ID
- 108: `claim_sets` references a claim whose `id` is missing
- 110: one credential's `claims` array repeats the same `id`
- 111: a claim has a present but empty `id`
- 112: a claim `id` contains a forbidden character
- 113: a claim object omits the required `path`
- 114: a claim `path` is an empty array
- 115: a claim `path` is not an array
- 120: a claim `path` array contains a Boolean
- 121: a claim `path` array contains a negative integer
- 122: a claim `path` is not an array; this duplicates case 115
- 123: a claim `path` array contains an unsupported element such as Boolean

The public `https://verifier-backend.eudiw.dev/ui/presentations` endpoint cannot
exercise these cases. Its typed request decoder rejects them before a signed
request reaches the Wallet:

- 096 fails because `CredentialSetQuery.options` is required.
- 097 and 098 fail during the same DCQL deserialization/validation boundary.
- 100 fails with `Unknown credential query ids`.
- 108 fails with `Unknown claim ids` in `ClaimSet.ensureKnownClaimIds`.
- 110 fails in `CredentialQuery.ensureUniqueIds` because a claim `id` is repeated.
- 111 fails in `ClaimId` validation with `Value cannot be be empty`.
- 112 fails in `DCQLId.ensureValid` because the claim `id` contains a character outside the allowed alphabet.
- 113 fails during `ClaimsQuery` decoding because the required `path` field is missing.
- 114 fails in `ClaimPath` deserialization because the path is empty.
- 115 fails during `ClaimPath` decoding because an array was expected.
- 120, 121, and 123 fail `ClaimPath` member validation before request creation.
- 122 fails during `ClaimPath` decoding for the same reason as case 115.

A direct by-value case 110 emulator probe reached the reference Wallet but
silently returned to Home without showing `invalid_request` or an error page.
This is not a passing outcome because case 110 explicitly requires an
`invalid_request` response. A response-capturing mock verifier is needed to
determine whether the Wallet sent that protocol error without rendering it.

Please provide a mock verifier or raw request generator that can serve a validly
signed Authorization Request containing deliberately malformed DCQL. It should
expose the request URI/deep link needed by the Wallet and record the resulting
error or privacy-preserving interaction discontinuation.

The mock service should support:

- request-object signing compatible with the reference Wallet;
- arbitrary DCQL JSON without deserializing it into a stricter server model;
- request URI and/or open-link delivery;
- captured Wallet response/error evidence;
- transaction correlation for Maestro and Credimi pipelines.

Without this service, these cases can only be statically validated and cannot be
claimed as device-level Wallet conformance tests through the public verifier.
```

### Issue 19: Provide controllable verifier response and error behavior

Suggested title:

```text
test(fcaf): add configurable verifier response harness
```

Suggested issue body:

```markdown
FCAF cases 124-134, 136-145 cannot be implemented against
the public reference verifier because they require behavior that its normal
presentation endpoint does not expose:

- 124: preserve an unknown parameter in the signed Authorization Request. A
  live probe showed the public endpoint silently strips the submitted field.
- 125: return HTTP 200 with a plain-text, non-JSON body after Wallet submission.
- 126: return HTTP 400 with a JSON body after Wallet submission.
- 127: return HTTP 200 JSON containing an unknown response parameter.
- 128: combine an unknown signed request parameter with an unknown response
  parameter.
- 129-132: retain the raw compact JWE so its protected `kid`, `enc`, and payload
  structure can be validated. The public poll endpoint returns only the
  decrypted Wallet response transfer object.
- 133-134: record the Wallet's HTTP method, content type, and exact form body at
  the response_uri.
- 136: send unsupported transaction_data and capture the Wallet error without
  entering credential selection.
- 137-139: send unknown, malformed, and empty scope values and capture an exact
  invalid_scope response.
- 140: repeat the empty-scope error and additionally prove the Wallet terminates
  the session without further communication.
- 141-145: serve each conflicting or invalid Authorization Request and capture
  the exact invalid_request or invalid_client response.

Please provide a transaction-correlated mock verifier that can create and sign
custom Authorization Requests, configure response_uri HTTP status, content
type, and body, capture Wallet protocol responses, and expose session-lifecycle
events. UI return to Home is not sufficient evidence for these cases because
their expected results require exact errors or response-processing behavior.

Until that harness exists, these cases are intentionally marked missing in the
Credimi implementation inventory. They are skipped implementations, not passing
tests and not accepted interaction discontinuations.
```

### Issue 20: Define the supported transaction-data fixture for test 135

Suggested title:

```text
clarify(wallet-rp): specify transaction data fixture for protocol test 135
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages_135` requires the Wallet to display and approve
transaction data, then bind it through `transaction_data_hashes`, but the test
does not identify a transaction-data `type`, required fields, or credential
fixture that the Wallet is expected to support.

This omission changes the semantics materially: an unsupported type belongs to
negative case 136. The current Android wallet core 0.28.1 explicitly rejects
all non-empty transaction_data before credential selection, so an arbitrary
test object cannot exercise the positive case.

Please define a normative or suite-local supported transaction-data object,
the matching requested credential, the UI values that must be displayed, and
the canonical bytes/hash algorithm used to verify transaction_data_hashes.
Until then, case 135 is not portable and is marked missing rather than being
implemented with an invented vendor-specific type.
```

### Issue 14: Expose transaction diagnostics for successful request shapes

Suggested title:

```text
test(verifier): expose transaction diagnostics when Wallet returns no presentation
```

Suggested issue body:

```markdown
For valid DCQL requests in FCAF cases 105, 109, 116, and 147, the reference
Wallet accepts the request and PIN, then returns to Home without showing the
consent or share screen. Case 109 specifically omits each claim `id` while also
omitting `claim_sets`, as permitted by OID4VP section 6.3. The public verifier
accepts this request shape, so it is not blocked by the malformed-request
limitation affecting cases 096-098, 100, and 108.

Case 116 requests a claim with a valid `path` while omitting the optional
`values` property. The public verifier also accepts this shape, but the Wallet
does not display consent or produce a presentation.

Case 147 requests a credential type absent from the Wallet. The Wallet displays
its local unavailable-document screen, but `Go Back` does not submit the
required `access_denied` error to the verifier.

After the case 109, 116, and 147 emulator attempts, the verifier transaction endpoint
returned HTTP 400 with an empty body. The failures therefore cannot be classified
as request rejection, credential mismatch, Wallet discontinuation, or a
request-routing failure.

Please expose structured transaction diagnostics including the Wallet response,
error code, error description, and lifecycle state. This is needed to distinguish
an invalid request, no matching credential, user cancellation, and transport or
callback failure in automated conformance evidence.
```

### Issue 15: Correct the expected error name in protocol test 114

Suggested title:

```text
fix(wallet-rp): correct invalid_request typo in protocol test 114
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages__114.md` says the Wallet returns an
`invalid request_error`. The OpenID4VP error value used by the neighboring tests
and required by this scenario is `invalid_request`.

Proposed change: replace `invalid request_error` with `invalid_request` and
format it as a protocol literal.
```

### Issue 16: Clarify the distinction between protocol tests 087 and 117

Suggested title:

```text
clarify(wallet-rp): distinguish multi-query tests 087 and 117
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages__087` and `WS_RP_MS_ProtocolMessages__117` can be
implemented with the same DCQL request: two credential query objects, both
matching one stored PID, with `credential_sets` omitted.

Test 087 focuses on whether two separate queries may be satisfied by the same
stored credential. Test 117 focuses on the default AND semantics when
`credential_sets` is absent: every credential query is mandatory. The current
scenarios do not state what evidence distinguishes these obligations.

Proposed changes:

- In 087, explicitly require both response entries to be derived from the same
  stored credential and define how that fact can be evidenced.
- In 117, explicitly require two distinct query IDs, omission of
  `credential_sets`, both query IDs in the `vp_token`, and one consent bundle
  that includes both requested entries.
- State whether 117 requires two distinct stored credentials or only two
  credential query objects. The current wording alternates between
  "credentials" as query objects and Credentials stored in the Wallet.

Observed reference-wallet behavior for 117 was one consent screen containing
two PID rows for two distinct query IDs. Expanding one row expanded both rows,
and each exposed the requested `Given Name(s)` claim. One Share/PIN interaction
produced a successful response containing both query IDs.
```

### Issue 17: Provide a resolvable fixture for all claim path component types in test 118

Suggested title:

```text
clarify(wallet-rp): define resolvable claim paths for protocol test 118
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages__118` says a claims path pointer is a non-empty
array whose members include strings, nulls, and non-negative integers. It is
unclear whether one path must contain all three types or whether the test must
cover the three allowed member types across separate resolvable paths.

A normal PID does not expose a natural nested claim for which one sequential
path containing all three types resolves successfully. For example,
`["nationality", null]` and `["nationality", 0]` are independently valid, but
combining both selectors in one path would attempt to apply the final selector
to a scalar nationality value.

Proposed change: provide an explicit credential fixture and exact path, or state
that separate resolvable paths may collectively cover string, null, and
non-negative integer components. The Credimi implementation uses
`["given_name"]`, `["nationality", null]`, and `["nationality", 0]` in one DCQL
credential query.

Observed with the reference Android Wallet: the public verifier accepted this
request, the Wallet reached consent, and sharing completed. A fresh PID was
issued for a dedicated Keycloak user whose realm profile was verified to
contain `nationality: ["IT"]`, birth date, and structured birthplace. The
Wallet offered both that PID and an older PID, but expanding both consent rows
showed only `Given Name(s)`. The HTTP 200 verifier response contained two
presentations whose only disclosures were `given_name: FCAF` and
`given_name: Filippo`; neither disclosed `nationality`. This provides evidence
of a Wallet path-resolution limitation, while an explicit fixture is still
needed to remove ambiguity about whether all component types belong in one
path or may be covered across separate paths.
```

### Issue 18: Deduplicate empty claim path tests 114 and 119

Suggested title:

```text
test(wallet-rp): deduplicate protocol tests 114 and 119
```

Suggested issue body:

```markdown
`WS_RP_MS_ProtocolMessages__114` and `WS_RP_MS_ProtocolMessages__119` both send
a DCQL claim object whose `path` is an empty array and require rejection. Test
114 specifically expects `invalid_request`; test 119 permits `invalid_request`,
an unspecified error, or interaction discontinuation.

The request shape is identical, while the accepted outcomes differ. This can
produce contradictory verdicts for the same Wallet behavior and duplicate
implementation artifacts.

Proposed change: merge the tests and choose one normative outcome, or explain
the distinct protocol context that makes their expected results differ.
```
