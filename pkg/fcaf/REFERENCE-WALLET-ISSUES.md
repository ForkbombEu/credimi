<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Reference Wallet Issues

## RI-WALLET-001: Empty DCQL credential_sets options is not returned to the verifier

### Summary

The reference Android wallet detects a DCQL `credential_sets[].options` array
that is present but empty and displays an error page, but it does not return
the required `invalid_request` response to the verifier.

### Reproduction

1. Create an OpenID4VP session with `response_mode: direct_post` at
   `https://beta-capture-wallet.credimi.io/openid4vp/sessions`.
2. Send a signed request containing:

```json
{
  "dcql_query": {
    "credentials": [
      {
        "id": "pid",
        "format": "dc+sd-jwt",
        "meta": { "vct_values": ["urn:eudi:pid:1"] }
      }
    ],
    "credential_sets": [{ "options": [] }]
  }
}
```

3. Unlock the wallet and open the generated deeplink.

### Observed

- The wallet briefly displays `Oups! Something went wrong` with
  `options cannot be empty`.
- The beta capture session records `vp_request_retrieved` only.
- No POST reaches the advertised `response_uri`; therefore no
  `error=invalid_request` is captured.

### Expected

The wallet must send an OpenID4VP `direct_post` error response with
`error=invalid_request`, in addition to the user-visible error page.

### Control

The same capture verifier successfully recorded the reference wallet's
`invalid_request` response for the adjacent missing-options case
`WS_RP_MS_ProtocolMessages__096`, so the capture endpoint accepts this error
response class.

### FCAF Impact

- Test: `WS_RP_MS_ProtocolMessages__097`
- The Credimi test requires both the exact protocol error and a strict error
  page; the reference wallet currently fails the protocol assertion.

### Same defect for other credential_sets refusals

Re-checked on 13/10/2026 against production Capture with a fresh PID in the
wallet and claim-bearing queries. The wallet shows an error and records only
`vp_request_retrieved` for:

- `WS_RP_MS_ProtocolMessages__097`: `options cannot be empty` (unchanged).
- `WS_RP_MS_ProtocolMessages__100`: an option referencing an unknown
  credential query id (`Unknown credential query ids: [unknown_credential_id]`).
- `WS_RP_IA_MainInteraction__034`: a `required: true` set whose option needs a
  credential the wallet lacks (`The requested document is not available`).

Added 14/10/2026, from run `fcaf_assessment_4t9hxzgxlf.json`:

- `WS_RP_MS_ProtocolMessages__147`: the request asks for a credential the wallet
  cannot hold and the source requires `error=access_denied`; the wallet shows
  its unavailable-document screen and `Go Back`/`Cancel` posts nothing. The
  session records no error (confirmed again after the test moved to
  `access_denied_required`).
- `WS_RP_MS_ProtocolMessages__015`: no `invalid_request` for a non-array
  `dcql_query.credentials`; the session records no error and no `vp_token`
  [INFERENCE: the converted test's failure is the missing error response, not a
  wrong code].

Other DCQL refusals in the same run were posted as `invalid_request`
(ProtocolMessages 058, 061–063, 065, 066, 068, 069, 098), so the verifier
accepts the response class.

## MOCK-VERIFIER-001: Beta capture verifier rejects valid PID response on certificate SAN

For `WS_RP_MS_ProtocolMessages__099`, the reference wallet reaches the request,
shares the PID, and submits a `vp_token`. The beta capture verifier marks the
presentation invalid because the PID `iss` URI does not match a SAN-URI or
SAN-DNS entry in the issuer certificate.

Observed verifier event:

```text
pid[0]: The 'iss' claim in the payload does not match a 'SAN-URI' name and the
domain extracted from the HTTPS URI does not match a 'SAN-DNS' name in the x5c
certificate.
error: invalid_request
error_description: One or more presentations failed verification.
```

The wallet-side positive flow is evidenced through `Share`, PIN approval, and
the submitted `vp_token`. Verifier acceptance is blocked by certificate/SAN
validation and must not be reported as a wallet failure.

## RI-WALLET-002: Client Identifier Prefix rejections are not returned to the verifier

### Summary

The reference Android wallet rejects requests whose Client Identifier Prefix it
does not support, or that violate the prefix's rules, and displays an error
page, but it does not return the required `invalid_request` response to the
verifier.

### Reproduction

Create an OpenID4VP session at
`https://beta-capture-wallet.credimi.io/openid4vp/sessions` with one of the
`scenarios/fcaf-wallet-solution-relying-party-client-id-prefix-controls.yaml`
request bodies, unlock the wallet, and open the generated deeplink:

- a signed Request Object with `client_id` `redirect_uri:<response_uri>`;
- an `origin:` Client Identifier outside the Digital Credentials API;
- an unsupported `fcaf_unsupported_prefix:` Client Identifier;
- plain unsigned URL parameters with an `https://` Client Identifier.

### Observed

Wallet 2026.09.42, 28/09/2026:

- The wallet retrieves the Request Object where one is offered, then logs
  `Invalid resolution: UnsupportedClientIdPrefix`, or
  `InvalidClientIdPrefix(value=Invalid client_id: 'ORIGIN' cannot be used as a
  Client ID prefix)` for `origin:`.
- It displays `Oups! Something went wrong`.
- The beta capture session records at most `vp_request_retrieved`; no POST
  reaches the response endpoint, so no `error=invalid_request` is captured.

### Expected

OpenID4VP 1.0 Section 8.5 lists an unsupported Client Identifier Prefix and a
violation of a prefix's requirements as `invalid_request` cases; the source
tests require an error response whose `error` is exactly `invalid_request`.

### Same defect for Client Identifier binding failures

The `scenarios/fcaf-wallet-solution-relying-party-client-id-binding-controls.yaml`
requests show the same behaviour on 28/09/2026: the wallet retrieves the
Request Object, logs `InvalidJarJwt(cause=ClientId not found in certificate's
subject alternative names)` for an `x509_san_dns:` Client Identifier absent
from the leaf, `InvalidJarJwt(cause=ClientId does not match leaf
certificate's SHA-256 hash)` for a wrong `x509_hash:` value, and
`UnsupportedClientIdPrefix` for `decentralized_identifier:`, then shows the
error page without posting `invalid_request`. A signed Request Object without
`x5c` and a plain request with a `redirect_uri:http://` Client Identifier end
on the same error page; the captured logcat window recorded no reason for
those two.

### FCAF Impact

- Tests: `WS_RP_MS_Metadata__110`, `WS_RP_MS_Metadata__126`,
  `WS_RP_MS_Metadata__130`, `WS_RP_MS_Metadata__133`,
  `WS_RP_MS_ProtocolMessages__039`, `WS_RP_MS_ProtocolMessages__143`,
  `WS_RP_MS_ProtocolMessages__144`, `WS_RP_SM_RpIntegrity__007`,
  `WS_RP_SM_RpIntegrity__014`.
- The request-shape and visual-evidence assertions pass; the reference wallet
  fails the protocol assertion.

## RI-WALLET-003: Malformed client_metadata is not returned to the verifier

### Summary

The reference Android wallet rejects a signed Request Object whose
`client_metadata` is not a JSON object and displays an error page, but it does
not return the required `invalid_request` response to the verifier.

### Reproduction

Create an OpenID4VP session at
`https://beta-capture-wallet.credimi.io/openid4vp/sessions` with the
`create-non-object-client-metadata` body of
`scenarios/fcaf-wallet-solution-relying-party-metadata-request-controls.yaml`
(`request_mutation.request_object.set` `/client_metadata` to a string), unlock
the wallet, and open the generated deeplink.

### Observed

Wallet 2026.09.42, 28/09/2026:

- The wallet retrieves the Request Object by POST and logs `Unexpected JSON
  token ... Expected start of the object '{', but had '"' instead at path:
  $.client_metadata`.
- It displays `Oups! Something went wrong`.
- The beta capture session records `vp_request_retrieved` only; no
  `error=invalid_request` reaches the response endpoint.

### Expected

OpenID4VP 1.0 Section 5.1 requires `client_metadata` to be a JSON object; the
source test requires the Wallet to reject the request with `invalid_request`.

### FCAF Impact

- Test: `WS_RP_MS_Metadata__106`.
- The delivered-request and visual-evidence assertions pass; the reference
  wallet fails the protocol assertion.

## RI-WALLET-004: Authorization Response encrypted with A128CBC-HS256 instead of GCM

### Summary

The reference Android wallet encrypts the `direct_post.jwt` Authorization
Response with `enc: A128CBC-HS256` when the verifier also offers `A128GCM` and
`A256GCM`.

### Reproduction

Run `scenarios/fcaf-wallet-solution-relying-party-dcql-session-encryption.yaml`
against `https://capture-wallet.credimi.io`. On 08/10/2026 its Request Object
carried the generated `client_metadata.encrypted_response_enc_values_supported`
`[A128GCM, A256GCM, A128CBC-HS256]` with one `ECDH-ES` P-256 key.

### Observed

Pipeline run reported on 08/10/2026: the captured compact JWE in
`observed.wallet_response.value.response` has protected header
`enc: A128CBC-HS256`.

### Expected

The source test requires `enc` to be `A256GCM` or `A128GCM`. HAIP 1.0 Section 5
requires Wallets to support `A128GCM` or `A256GCM` and says a Wallet supporting
both SHOULD use `A256GCM`. OpenID4VP allows any advertised value, so the choice
is not a protocol error, but it is not the algorithm the source expects.

### FCAF Impact

- Test: `WS_RP_SM_SessionEncryption__001b` fails on
  `content_encryption_is_a256gcm_or_a128gcm`. The assertion is correct and the
  verifier list is left as Capture generates it, so the result reflects the
  wallet's own choice.
- `WS_RP_SM_SessionEncryption__007`–`009` advertise a single GCM value or only
  the two GCM values in dedicated scenarios, so they test whether the wallet
  honours a narrowed list, not this default choice.

## RI-WALLET-005: Two queries satisfied by one credential end in "Credential not found"

### Summary

When two DCQL credential queries match the same credential type, the reference
Android wallet lists both on the consent screen, accepts Share and the PIN, then
shows `Oups! Something went wrong / Credential not found` and posts nothing.

### Reproduction

13/10/2026, production Capture, emulator, one and then two freshly issued
SD-JWT VC PIDs: run
`scenarios/fcaf-wallet-solution-relying-party-dcql-same-credential-multiple-queries.yaml`
(two `dc+sd-jwt` PID queries for `given_name`) or
`scenarios/fcaf-wallet-solution-relying-party-dcql-claims-union.yaml`
(`given_name` and `address` in two PID queries).

### Observed

The session records only `vp_request_retrieved`; the wallet shows `Credential
not found` after the PIN.

### Expected

`WS_RP_MS_ProtocolMessages__087`: the wallet recognises that one credential
satisfies both queries and "MUST NOT return an error". `WS_RP_IA_MainInteraction__014`:
the wallet presents the union of both queries from the same credential.

### FCAF Impact

- Tests: `WS_RP_MS_ProtocolMessages__087`, `WS_RP_IA_MainInteraction__014`.
- Two queries for distinct credentials (SD-JWT VC and mdoc PID, as in
  `WS_RP_IA_MainInteraction__039` and `WS_RP_MS_ProtocolMessages__117`) succeed.

## RI-WALLET-006: Denied consent and failed user authentication are not returned to the verifier

### Summary

After the End-User denies consent, or enters a wrong PIN at the authorization
step, the reference Android wallet returns to its own screens without posting an
`access_denied` error response.

### Reproduction

13/10/2026, production Capture, emulator: run
`scenarios/fcaf-wallet-solution-relying-party-dcql-user-denied-consent.yaml` and
`scenarios/fcaf-wallet-solution-relying-party-dcql-user-authentication-failed.yaml`.

### Observed

Both sessions stay at `request_retrieved`; no POST reaches `response_uri`.

### Expected

An Authorization Error Response with `error=access_denied` and no `vp_token`.

### FCAF Impact

- Tests: `WS_RP_MS_ProtocolMessages__148`, `WS_RP_MS_ProtocolMessages__149`.
  Their consent and denial screenshots are captured; only the protocol
  assertion fails.

## RI-WALLET-007: Boolean-like strings accepted for require_cryptographic_holder_binding

### Summary

The reference Android wallet presents a credential when a DCQL credential query
sets `require_cryptographic_holder_binding` to the strings `"true"`, `"false"`,
`"TRUE"` or `"FALSE"`, instead of rejecting the non-boolean value.

### Reproduction

13/10/2026, production Capture, emulator: run
`scenarios/fcaf-wallet-solution-relying-party-dcql-holder-binding-type-boundary.yaml`.

### Observed

The four sessions above reach `presentation_validated`. Numbers (`0`, `1`,
random) and other strings (`"yes"`, `"no"`, `"y"`, `"n"`, `"0"`, `"1"`,
random) are rejected.

### Expected

`WS_RP_MS_ProtocolMessages__081`: the property is a boolean; any other type makes
the query invalid and the wallet rejects it.

### FCAF Impact

- Test: `WS_RP_MS_ProtocolMessages__081` fails on the four
  `*_no_presentation_returned` assertions for those strings.

Extended 14/10/2026: after the per-variant split of
`scenarios/fcaf-wallet-solution-relying-party-dcql-holder-binding-type-boundary.yaml`
(each variant is now its own wallet step with its own session and screenshots),
081 fails only through these four variants — the strings `true`, `false`,
`TRUE`, `FALSE`. The numeric values (`0`, `1`, random) and the other strings
(`yes`, `no`, `Y`, `N`, `0`, `1`, random) are rejected and pass.

## RI-WALLET-008: Plain direct_post response mode is refused

### Summary

The reference Android wallet refuses every Authorization Request with
`response_mode: direct_post`, whatever the scheme (`haip-vp://` or
`openid4vp://`), with `HAIP profile requires an encrypted response mode
(direct_post.jwt or dc_api.jwt)`.

### Reproduction

13/10/2026, production Capture, emulator: run
`scenarios/fcaf-wallet-solution-relying-party-supportive-redirect-uri.yaml`, or
create any session with `response_mode: direct_post`.

### Observed

The wallet shows the error above and posts nothing; the session stays at
`request_retrieved`.

### Expected

`WS_RP_IA_Supportive__001` sends `direct_post` with a `response_uri` and expects
the wallet to POST the Authorization Response and follow the returned
`redirect_uri`.

### FCAF Impact

- Test: `WS_RP_IA_Supportive__001` fails for this wallet. With
  `direct_post.jwt` the same wallet posts and follows the `redirect_uri`, but
  that is not the response mode the source specifies.

Confirmed on the emulator 14/10/2026 (wallet 2026.09.42, production Capture):
after a successful share the Wallet opens the returned `redirect_uri` only once
the user taps `Close` on its terminal screen; with the dismissal-enabled flow
Capture recorded `redirect_uri_visit_count: 1` for the callback-redirect session
(`WS_RP_IA_MainInteraction__057`, `061`, `067`; `WS_RP_MS_ProtocolMessages__127`,
`128` use the same pattern). The 4t9hxzgxlf run recorded no visit because its
flow never dismissed that screen. See RI-WALLET-023 for the HTTP 400 error reply
(`064`), whose `redirect_uri` the Wallet never opens.

## MOCK-VERIFIER-002: Beta signs decentralized_identifier requests with the X.509 key

On 28/09/2026 a beta session with `client_id_scheme: decentralized_identifier`
and no `request_behavior` delivered a Request Object whose `client_id` was
`decentralized_identifier:did:web:beta-capture-wallet.credimi.io:openid4vp` but
whose JOSE header carried `kid: credimi-fake-verifier-key` and the X.509
verifier `x5c`. The signature verified against that `x5c` leaf and not against
the only verification method in `/openid4vp/did.json`
(`#credimi-fake-verifier-did-key`). The published contract says this scheme
signs with a separate `did:web` key.

### FCAF Impact

- `WS_RP_IA_Metadata__015`, `016` and `WS_RP_SM_RpIntegrity__006` require the
  DID-published key to verify the request, so beta cannot satisfy them.
- `WS_RP_SM_RpIntegrity__007` still holds its precondition, because
  `request_behavior.signing_key: unrelated` also signs with a key the DID
  document does not list, but on beta the normal DID request would satisfy it
  too. The reference wallet additionally rejects `decentralized_identifier:`
  as an unsupported prefix, so it never reaches the signing-key check.

## MOCK-VERIFIER-003: Beta signed a configured redirect_uri beside response_uri (resolved)

On 30/09/2026 a beta `direct_post.jwt` session created with a top-level
`redirect_uri` delivered a Request Object carrying both `response_uri` and
`redirect_uri` (the configured URI plus its fresh `response_code`). OpenID4VP
1.0 Section 8.2: "When the response_uri parameter is present, the redirect_uri
Authorization Request parameter MUST NOT be present", and the Wallet must answer
`invalid_request`. Wallet 2026.09.42 did exactly that:

```text
error: invalid_request
error_description: RedirectUriMustNotBeProvided
```

The behaviour came from upstream `a2fc6a0`, synced on 28/09/2026.

### Resolution

Fixed on beta later on 30/09/2026: the configured `redirect_uri`, with its
`response_code`, is returned at session creation and in the Response Endpoint
reply, and is no longer signed into the Request Object. A probe session
delivered `response_uri` without `redirect_uri`.

### FCAF Impact

- `WS_RP_IA_MainInteraction__064` no longer needs a `request_mutation`
  workaround, and still asserts that the delivered request omits
  `redirect_uri`.
- `WS_RP_IA_MainInteraction__057`, `061`, `067`, `WS_RP_IA_Supportive__001`,
  and `WS_RP_MS_ProtocolMessages__127`, `128` are no longer rejected for this
  reason. None has a reference-Wallet run since the fix.

## RI-WALLET-009: A malformed Token Status List reference is not rejected

### Summary

When the issuer embeds a malformed `status` structure in a credential, the
reference Android wallet accepts and stores the credential instead of rejecting
it, and never shows its no-match screen for a probe query that targets it.

### Reproduction

Run `fcaf_assessment_4t9hxzgxlf.json` (13/10/2026, production Capture,
`https://capture-wallet.credimi.io`): issue each malformed `status_reference`
with `status_list_enabled: true`, offer the credential, then probe with a DCQL
query for the issued `document_number`.

| Test | `status_reference` | fixture | probe |
| --- | --- | --- | --- |
| `WS_RP_MS_Metadata__084` | `status_without_status_list` | `pid_family_name_uppercase` | `document_number = CREDIMI-DEMO-CASE` |
| `WS_RP_MS_Metadata__086` | `negative_index` | `pid_family_name_trailing_space` | `CREDIMI-DEMO-SPACE` |
| `WS_RP_MS_Metadata__087` | `missing_index` | `pid_locality_diacritics` | `CREDIMI-DEMO-UMLAUT` |
| `WS_RP_MS_Metadata__089` | `malformed_uri` | `pid_locality_no_diacritics` | `CREDIMI-DEMO-NOUMLAUT` |
| `WS_RP_MS_Metadata__090` | `missing_uri` | `pid_multiple_nationalities` | `CREDIMI-DEMO-ARRAY` |
| `WS_RP_MS_Metadata__092`/`__096`/`__097`/`__099`/`__100` | the same five shapes as mdoc (`urn:eu.europa.ec.eudi:pid:1.mdoc.key-attestation-required`) | — | `doctype`/namespace `eu.europa.ec.eudi.pid.1` |

### Observed

- The offer flow ends on the wallet's "Document issued" success screen — the
  screenshot
  `status_reference_rejection_..._probe_status_without_status_list_issue_pid_credential_added_*.png`
  shows the stored credential card with `View details` — so the flow's
  `assertNotVisible: "View details"` is what fails.
- The stored credential is then offered for the probe query. In
  `WS_RP_MS_Metadata__100`'s probe record the session shows
  `status: request_retrieved`, `observed.wallet_response.value: null` and
  `checks.errors: []`; the protocol probe passes vacuously on the empty session,
  and the failing assertion is the missing rejection evidence
  (`evidence value is empty`) [INFERENCE: the run's evidence is consistent with
  "the wallet kept the credential", not with a rejected issuance].

### Expected

The source requires the Wallet to reject a credential whose status reference
cannot be resolved, and to present none of it for the probe. Capture applied
every malformed `status_reference` (both formats), so no service gap blocked
these tests.

### FCAF Impact

- Tests: `WS_RP_MS_Metadata__084`, `086`, `087`, `089`, `090`, `092`, `096`,
  `097`, `099`, `100`.
- Extended 14/10/2026: `WS_RP_MS_Metadata__102` — a COSE/ISO mdoc Referenced
  Token whose Mobile Security Object carries no `status` element at all
  (`status_list_enabled: false`, fixture `pid_family_name_uppercase`,
  `document_number` `CREDIMI-DEMO-CASE`) is accepted and stored the same way;
  the wallet shows the issuance success screen and offers the stored credential
  on a `document_number` probe instead of rejecting the issuance.
- Withdrawn 14/10/2026 (self-review): 102's statusless-mdoc fixture and probe
  were removed from the scenario and 102 reports `fcaf.skipped` for the unbuilt
  COSE/CWT Referenced Token again, so no test asserts the case above today; the
  observation comes from the probe as it stood in run `4t9hxzgxlf`.

## RI-WALLET-010: A DCQL request-evaluation error is shown but never returned

### Summary

For a request whose DCQL query is malformed, the reference Android wallet either
renders its own error page naming the defect or stops at its no-match screen,
and posts nothing: no `invalid_request` (or `access_denied`) Authorization Error
Response reaches the verifier.

### Reproduction

Run `fcaf_assessment_4t9hxzgxlf.json` (13/10/2026, production Capture): run the
`fcaf-wallet-solution-relying-party-dcql-*` scenarios whose signed `dcql_query`
carries one malformed member (claim `id`, claim `path`, claim `values`,
duplicate claim ids), unlock the wallet and open the deeplink.

### Observed

The wallet shows the defect on screen and the session records no error response:

- `WS_RP_MS_ProtocolMessages__107` — no-match screen
  (`The requested document is not available in your EUDI Wallet`), nothing
  posted; its source requires `error=access_denied`.
- `...__108` — `Oups! Something went wrong / Unknown claim ids`.
- `...__110` — `... / Within a CredentialQuery, the same id of claims MUST NOT be present more than once`.
- `...__111` — `... / Value cannot be be empty`.
- `...__112` — `... / The value must be a non-empty string consisting of alphanumeric, underscore (_) or hyphen (-) characters`.
- `...__114` — `... / Failed requirement`.
- `...__120` — `... / Only string, null, int can be used`.
- `...__121` — `... / Index should be non-negative`.
- `...__123` — the same UI error page, nothing posted.
- `WS_RP_SH_Encoding_TextualEncoding__007` — `... / Can only select from object or array of objects`.
- `...__009` — `Element class kotlinx.serialization.json.b… is not a JsonArray`.
- `...__010` — `Element class rr4 (Kotlin reflection is not available) is not a JsonArray`.
- `...__012` — `... / Only string, null, int can be used`.
- `...__018` — `... / ClaimPaths for MSO MDoc based formats must have exactly two elements`.
- `...__019` — `... / ClaimPaths for MSO MDoc based formats must contain only Claim ClaimPathElements`.
- `...__013`, `...__016`, `...__020` — no error text: `The requested document is not available in your EUDI Wallet` with `Share` disabled, then stop.
- `WS_RP_MS_ProtocolMessages__015` — no `invalid_request` for a non-array
  `dcql_query.credentials` [INFERENCE: the run records no error and no
  `vp_token`; the converted test's failure is the missing response].

In every case `raw.presentation_response_decrypted` is absent. Screenshot names
are listed in `fcaf-4t9hxzgxlf-failure-review.md`.

### Expected

The sources require the Wallet to reject the request with `invalid_request` (or,
for 107, `access_denied`) instead of stopping locally. The same Capture verifier
recorded real Wallet error responses for the sibling controls 058, 061–063, 065,
066, 068, 069, 096, 098, 113, 115 and 122, so the response class is accepted.

### FCAF Impact

- Tests: `WS_RP_MS_ProtocolMessages__015`, `107`, `108`, `110`, `111`, `112`,
  `114`, `120`, `121`, `123`; `WS_RP_SH_Encoding_TextualEncoding__007`, `009`,
  `010`, `012`, `013`, `016`, `018`, `019`, `020`.
- The request-shape assertions pass; the wallet fails the error-response
  assertion. Same mechanism as RI-WALLET-001 (credential_sets refusals and
  `147`), RI-WALLET-002 (Client Identifier Prefixes) and RI-WALLET-003
  (malformed `client_metadata`).

## RI-WALLET-011: A null array selector is not rejected on arrays of objects

### Summary

A DCQL claim path that selects inside an array of objects with an invalid
element (a `null` component) is not rejected: the wallet fails after the user
approves sharing and never answers the verifier.

### Reproduction

Run the degree-credential claims-path scenarios of
`WS_RP_SH_Encoding_TextualEncoding__002`, `__008` and `__011` against production
Capture (`fcaf_assessment_4t9hxzgxlf.json`, 13/10/2026).

### Observed

- `002`/`008`: `captured evidence does not contain vp_token`; no
  `…present_degree_credential…` screenshot exists for the session, while the
  issuance screenshot does.
- `011`: the `dcql_degree_index_selector_…_present_degree_credential_…png`
  screenshot shows the full `Data sharing request / Fake Verifier` consent
  screen with `Share` enabled, and the step's `visual_evidence` is empty
  [INFERENCE: the wallet stopped after `Share`, before posting anything].

No error response reaches the verifier in any of the three sessions.

### Expected

The source requires the selector to resolve or the request to be rejected with
an error response; a silent stop is neither.

### FCAF Impact

- Tests: `WS_RP_SH_Encoding_TextualEncoding__002`, `__008`, `__011`.

## RI-WALLET-012: A non-URL-safe nonce is accepted and a credential is presented

### Summary

The reference Android wallet accepts an Authorization Request whose `nonce` is
not URL-safe and presents a credential, instead of rejecting the request with
`invalid_request`.

### Reproduction

13/10/2026, production Capture, emulator: run
`scenarios/fcaf-wallet-solution-relying-party-session-binding-invalid-nonce.yaml`
(`WS_RP_SM_SessionBinding__003`), whose signed Request Object carries the
verbatim nonce `fcaf nonce/!`.

### Observed

`wallet returned vp_token, expected error "invalid_request"` — the session
contains a non-empty `vp_token`. The precondition (the exact nonce in the
delivered request) passed on the same session.

### Expected

The source requires `invalid_request` without a presentation.

### FCAF Impact

- Test: `WS_RP_SM_SessionBinding__003`.

## RI-WALLET-013: An unusable verifier encryption JWK is not refused

### Summary

When the verifier's `client_metadata` publishes an encryption JWK the wallet
cannot use according to HAIP — an `alg` that is not bare `ECDH-ES`, or no bare
`ECDH-ES` key at all — the reference Android wallet does not return
`invalid_request`: it either answers anyway or stops silently.

### Reproduction

13/10/2026, production Capture, emulator, with
`allow_undecryptable_response: true`:

- `WS_RP_SM_SessionEncryption__003` —
  `fcaf-wallet-solution-relying-party-response-encryption-jwk-alg-mismatch`
  (`alg: ECDH-ES+A256KW`; HAIP pins the JWE `alg` to bare `ECDH-ES`, so the
  wallet must refuse rather than substitute it).
- `WS_RP_SM_SessionEncryption__006` —
  `fcaf-wallet-solution-relying-party-response-encryption-no-ecdh-es-jwk` (a
  single valid P-256 `use: enc` JWK with `alg: ECDH-ES+A256KW`, i.e. no bare
  `ECDH-ES` key).

### Observed

- `003`: `capture recorded 1 "vp_presentation_response_received" event(s),
  expected 0` **and** `wallet returned no error, expected "invalid_request"` —
  the wallet submitted a response the verifier rejected
  (`Oups! Something went wrong / Verifier rejected the response`), instead of
  refusing the malformed JWK.
- `006`: `wallet returned no vp_token but no error, expected error`, with the
  same `Verifier rejected the response` screen — the wallet stopped without
  posting an error response.

### Expected

OID4VP Section 8.3 requires `alg` in the published JWKs and the JWE `alg` to
equal it; the sources require `invalid_request` without a presentation.

### FCAF Impact

- Tests: `WS_RP_SM_SessionEncryption__003`, `WS_RP_SM_SessionEncryption__006`.
- Adjacent to RI-WALLET-004 (wrong content encryption chosen when the list is
  not narrowed).

## RI-WALLET-014: Request URI POST advertises a list of acceptable media types

### Summary

For a `request_uri_method: post` retrieval the reference Android wallet sends
`Accept: application/oauth-authz-req+jwt,application/jwt,application/json`
instead of the single `application/oauth-authz-req+jwt` the source pins.

### Reproduction

13/10/2026, production Capture: run
`scenarios/fcaf-wallet-solution-relying-party-request-uri-post.yaml`
(`WS_RP_MS_ProtocolMessages__042`).

### Observed

`oid4vp.request_uri_retrieval` fails with `request_uri retrieval Accept header
is "application/oauth-authz-req+jwt,application/jwt,application/json", expected
"application/oauth-authz-req+jwt" - method and Content-Type passed`.

### Expected

The source asks the Wallet to request the Request Object as a JWT. Whether
OpenID4VP requires `Accept` to carry that media type alone, and not a
preference list, is [INFERENCE]; this is the borderline entry of the run's
Wallet findings, and the assertion pins the exact header.

### FCAF Impact

- Test: `WS_RP_MS_ProtocolMessages__042` (its sibling 044, which checks UTF-8
  in the POST body, is unaffected).

## RI-WALLET-015: A SHA-384 credential is presented to a SHA-256-only verifier and the algorithm is not declared

### Summary

The reference Android wallet presents an SD-JWT VC digested with SHA-384 to a
verifier that advertises SHA-256 only, and its `wallet_metadata` does not name
`sha-384` as an algorithm it supports.

### Reproduction

13/10/2026, production Capture: issue the PID with `digest_algorithm: sha-384`
and present it to

- a verifier that advertises SHA-256 only (`WS_RP_SH_Cryptography_CryptographicHash_007`),
- the default session whose `wallet_metadata` is inspected for declared hash
  algorithms (`WS_RP_SH_Cryptography_CryptographicHash_006`).

### Observed

- `006`: the session reaches `presentation_validated`;
  `decoded_presentations.pid_sha384[0].digest_algorithm = "sha-384"`, while the
  wallet's `wallet_metadata` names no `sha-384`.
- `007`: `vp_token_holds_only_sha256_digested_credentials: presentation 0 is
  digested with sha-384, outside sha-256`; the
  `credential_digest_algorithm_…_exercise_sha256_only_presentation_…png`
  screenshot shows `Authentication successful – You successfully shared …
  Credimi Demo PID`.

### Expected

The source requires a Wallet that supports only SHA-256 not to present a
credential digested with another hash function, and a Wallet to declare the
digest algorithms it can process.

### FCAF Impact

- Tests: `WS_RP_SH_Cryptography_CryptographicHash__006`, `__007`.
- Capture advertises SHA-256 only in its Credential Issuer Metadata while
  issuing the `sha-384`-digested credential, which is what the source
  (`WS_RP_SH_Cryptography_CryptographicHash__010`) asks for; the wallet's own
  behaviour is the finding.

## RI-WALLET-016: The Authorization Error code does not match the source

### Summary

The reference Android wallet answers a malformed scope or format request with
`error=invalid_request` where the source requires a specific code: `invalid_scope`
for the unknown scope (139/140) and `vp_formats_not_supported` for the
unsupported format (150).

### Reproduction

13/10/2026, production Capture: run the unknown-scope and unsupported-format
scenarios of `WS_RP_MS_ProtocolMessages__139`, `__140` and `__150`.

### Observed

- `139`/`140`: `wallet returned error "invalid_request", expected "invalid_scope"`.
- `150`: `wallet returned error "invalid_request", expected "vp_formats_not_supported"`.
- 140's termination half passed, so the wallet does send an error response.

### Expected

Each source names its exact error code; a broader generic code is not the
required response.

### FCAF Impact

- Tests: `WS_RP_MS_ProtocolMessages__139`, `140`, `150`.

## RI-WALLET-017: `decentralized_identifier` is not supported

### Summary

The reference Android wallet does not process an Authorization Request whose
Client Identifier is a `decentralized_identifier:`, and answers nothing.

### Reproduction

13/10/2026, production Capture: run
`scenarios/fcaf-wallet-solution-relying-party-did-metadata.yaml`
(`WS_RP_SM_RpIntegrity__006`), which requests
`client_id_scheme: decentralized_identifier` and expects the Wallet to verify
the request with the key published in `/openid4vp/did.json`.

### Observed

`oid4vp.session_event_count` fails with `capture recorded 0
"vp_presentation_response_received" event(s), expected 1; observed events:
vp_session_created, vp_deeplink_generated, vp_request_retrieved`; the
`did_metadata_…_exercise_wallet_…png` screenshot shows the wallet's
`Oups! Something went wrong` page.

### Expected

The source requires the Wallet to accept the DID-based Client Identifier and
verify the JWS with the DID-documented key.

### FCAF Impact

- Test: `WS_RP_SM_RpIntegrity__006` (and the `decentralized_identifier` cases
  `WS_RP_IA_Metadata__015`, `__016`, recorded under MOCK-VERIFIER-002 as well).
- [INFERENCE] MOCK-VERIFIER-002 (beta signed `decentralized_identifier`
  requests with the X.509 key) may also contribute; the wallet's own
  unsupported-prefix refusal is documented in RI-WALLET-002, so a
  wallet-side `decentralized_identifier` gap is the accurate record for a
  service that signs correctly.

## RI-WALLET-018: Unsigned and multisigned DC API requests are never answered

### Summary

The reference Android wallet produces no Authorization Response and no DC API
invocation outcome for an unsigned (`openid4vp-v1-unsigned`) or a multisigned
(`openid4vp-v1-multisigned`) DC API request.

### Reproduction

13/10/2026, production Capture: run the unsigned and multisigned DC API
scenarios of `WS_RP_SM_RpIntegrity__028` and `WS_RP_SM_RpIntegrity__030`.

### Observed

- `028`: `capture recorded 0 "vp_presentation_response_received" event(s),
  expected 1; observed events: vp_session_created, vp_deeplink_generated`.
- `030`: the session "records neither an Authorization Response nor a DC API
  invocation outcome, so the wallet was never invoked"; the multisigned block
  produced only the `issue pid credential added` screenshot.

### Expected

The source requires the Wallet to process the request under the DC API protocol
and return the corresponding response/invocation outcome.

### FCAF Impact

- Tests: `WS_RP_SM_RpIntegrity__028`, `WS_RP_SM_RpIntegrity__030`.
- The DC API runner is demonstrably healthy: 7 of the 9 `pipeline.dc-api.*`
  tests pass in the same run, and the three working blocks have terminal
  `dc api invocation outcome` screenshots.

## RI-WALLET-019: A wrong audience and an untrusted certificate root are not rejected

### Summary

The reference Android wallet does not reject a signed request whose `aud` does
not match its Client Identifier, and does not reject a request whose `x5c`
chain ends in an untrusted root: it proceeds to the consent screen.

### Reproduction

13/10/2026, production Capture: run the `aud`-mismatch scenario of
`WS_RP_IA_Metadata__013` and the untrusted-root scenario of
`WS_RP_SM_RpIntegrity__019`.

### Observed

- `013`: `wallet returned no error, expected "invalid_request"`; the
  `…reject_aud_mismatch_expect_request_rejected_….png` screenshot shows the
  wallet on its `Data sharing request` consent screen (spinner, `Cancel`/`Share`).
- `019`: `wallet returned no error, expected "invalid_request"`; the
  `…reject_untrusted_root_expect_request_rejected_….png` screenshot shows the
  same consent screen, i.e. the presentation flow was initiated.

### Expected

Both sources require `invalid_request` and no presentation.

### FCAF Impact

- Tests: `WS_RP_IA_Metadata__013`, `WS_RP_SM_RpIntegrity__019`.
- Same family as RI-WALLET-002 ("rejection not returned"), but here the wallet
  does not reject at all.
- Extended 14/10/2026: `WS_RP_SM_RpIntegrity__016` and `WS_RP_SM_RpIntegrity__018`
  keep a `fcaf.skipped` assertion for their chain-validation half, because
  Capture serves a single-element, leaf-only `x5c`. This issue is the reason a
  complete chain would not settle that property either on this Wallet, which
  accepts a request whose chain ends in an untrusted root. The rationale was
  moved out of 016's skip reason, which must name only the unbuilt precondition.

## RI-WALLET-020: A claimless DCQL credential query is refused

### Summary

The reference Android wallet refuses a DCQL credential query that carries no
`claims` member, even when it holds a matching credential, and posts no
response.

### Reproduction

13/10/2026, production Capture: run
`scenarios/fcaf-wallet-solution-relying-party-dcql-claims-omitted.yaml` and open
the deeplink for `WS_RP_IA_MainInteraction__016`, `WS_RP_IA_MainInteraction__017`
(mdoc form) and `WS_RP_MS_ProtocolMessages__082`.

### Observed

Every session records the wallet's `The requested document is not available in
your EUDI Wallet` screen with `Share` disabled and no POST to the verifier
(`wallet vp_token is not an object keyed by credential query ID`).

### Expected

`WS_RP_MS_ProtocolMessages__082` requires a credential-bearing response for the
query id; `WS_RP_IA_MainInteraction__016` requires an SD-JWT presentation with
zero disclosures; `__017` the same for an mdoc.

### FCAF Impact

- Tests: `WS_RP_IA_MainInteraction__016`, `WS_RP_IA_MainInteraction__017`,
  `WS_RP_MS_ProtocolMessages__082`.
- Documented in `pkg/fcaf/MEMORY.md` ("Claimless DCQL queries the sources do not
  require") before this entry existed; the maintainers kept the four claimless
  tests because the sources require them.

## RI-WALLET-021: A present-but-empty `client_metadata`/`meta` is refused

### Summary

The reference Android wallet refuses a DCQL credential query for `dc+sd-jwt`
whose `meta` object is present but empty, requiring `vct_values` even when the
`meta` object carries no other member.

### Reproduction

13/10/2026, production Capture: run the empty-`meta` scenario of
`WS_RP_MS_ProtocolMessages__076` and open the deeplink.

### Observed

- The wallet shows `Oups! Something went wrong / Field 'vct_values' is required
  for type with serial name 'eu.europa.ec.eudi.openid4vp.dcql.DCQLMetaSdJwtVcExtensions',
  but it was missing`; nothing is posted.
- The earlier validator rejected the same query itself, so the wallet's
  behaviour only became visible after the 14/10/2026 fix that accepts a
  present-but-empty `meta` for `dc+sd-jwt`.

### Expected

The source's form ("no further constraints") must be accepted and answered with
a credential; `claims` and `meta` are independently optional.

### FCAF Impact

- Test: `WS_RP_MS_ProtocolMessages__076`.

## RI-WALLET-022: `trusted_authorities` is ignored when matching credentials

### Summary

The reference Android wallet offers and presents a PID whose issuer chain does
not match the request's `trusted_authorities`.

### Reproduction

14/10/2026, emulator, wallet 2026.09.42, production Capture: issue a PID from
Capture (issuer AuthorityKeyIdentifier `QlBQvhC4EPCdRFyNv6sQCO4n3Ek`), then open
the `create-no-match-session` request of
`scenarios/fcaf-wallet-solution-relying-party-dcql-trust-mechanisms.yaml`, whose
query carries `trusted_authorities: [{type: aki, values:
["AAAAAAAAAAAAAAAAAAAAAAAAAAA"]}]`.

### Observed

- The wallet shows `Data sharing request` with the PID under `Requested data`
  instead of its no-match notice, and shares it.
- Capture records `presentation_invalid` with Credo's `Expected one of the
  trusted authority values to be 'AAAAAAAAAAAAAAAAAAAAAAAAAAA' but received
  'QlBQvhC4EPCdRFyNv6sQCO4n3Ek'`.
- Consistent with the earlier observation that the wallet answers malformed
  `trusted_authorities` members (MEMORY.md, reference wallet 2026.09.42 table).

### Expected

No credential is offered; the wallet informs the user it holds no matching
credential (OpenID4VP 6.1.1).

### FCAF Impact

- Tests: `WS_RP_SM_TrustMechanisms__004`, `__008`,
  `WS_RP_IA_MainInteraction__004`. Before credimi-capture-wallet#10 was fixed
  (upstream `2422e58`) Capture replaced the shared `vp_token` with its own
  error, so these tests passed falsely.

## RI-WALLET-023: `redirect_uri` in an HTTP 400 response-endpoint reply is not followed

### Summary

When the Verifier's response endpoint answers the Authorization Response with an
HTTP error carrying a `redirect_uri`, the reference Android wallet shows its error
screen and never opens that URI.

### Reproduction

14/10/2026, emulator, wallet 2026.09.42, production Capture: the
`create-http-error-redirect-response` session of
`scenarios/fcaf-wallet-solution-relying-party-verifier-response-controls.yaml`.

### Observed

- After sharing, the wallet shows `Oups! Something went wrong / Verifier
  rejected the response` with `TRY AGAIN` and a close control; closing it
  returns to the `Data sharing request` screen.
- Capture records no `vp_redirect_uri_visited` event.

### Expected

The wallet navigates the user agent to the returned `redirect_uri`, as
`WS_RP_IA_MainInteraction__064` requires.

### FCAF Impact

- Test: `WS_RP_IA_MainInteraction__064`.

## RI-WALLET-024: A DCQL request offers no per-claim deselection

### Summary

The reference Android Wallet renders the consent screen of a DCQL (or DC API)
presentation request with no per-claim deselection control, so the user cannot
withhold a single requested claim. The sources that require a user to unselect
a claim, such as `WS_RP_IA_MainInteraction__013`, therefore cannot be
satisfied by this SUT.

### Reproduction

14/10/2026, emulator, Wallet 2026.09.42, production
`capture-wallet.credimi.io`: run the `unchecked-claim` session of
`scenarios/fcaf-wallet-solution-relying-party-dcql-unchecked-claim.yaml`
(`WS_RP_IA_MainInteraction__013`), open the deeplink, unlock, and try to
unselect the `address` claim on the consent screen.

### Observed

- The consent screen lists the requested data read-only; tapping the `address`
  group only expands it and no checkbox is offered.
- The run's `fcaf-013-unchecked-address` screenshot shows the expand-only row,
  and the Wallet discloses `address` in the submission.

### Expected

`WS_RP_IA_MainInteraction__013` (source l.33): "User unselects `address` claim
in the UI, that claim is absent in the vp_token".

### Notes

`[INFERENCE]` At the Wallet's 2026.09.42 tag, `WalletCorePresentationController.kt`
(279-282) returns `requestAllowsClaimSelection = false` for `ProcessedDcqlRequest`
and `ProcessedOpenId4VpDCAPIRequest` ("these protocols validate the selection as
a complete set"), so the consent screen renders read-only claim rows and
`RequestScreen.kt` passes a null item callback. The screenshot and the app's
test tags agree with that reading, but the source line was not re-read from the
APK in this session.

### FCAF Impact

- Test: `WS_RP_IA_MainInteraction__013`, which now asserts
  `sdjwt.claim_presence{address, present: false}` and
  `{given_name, present: true}` on the Wallet's own `wallet_response`. It
  FAILS on this Wallet.
- Maintainer decision required: keep the strict failure (the source relevancy is
  `EUDI_generic | EUDI_required`, so a missing capability is arguably a breach)
  or report `fcaf.not_applicable` per the `.agents/HITL.md` entry of
  2026-10-14 ("not_applicable when the SUT cannot meet a source precondition").

## RI-WALLET-025: A malformed response-endpoint reply does not produce a Wallet error screen

### Summary

After the Wallet has submitted its Authorization Response, a reply from the
Verifier's response endpoint that carries no `redirect_uri`
(`WS_RP_IA_MainInteraction__066`) or whose HTTP 200 body is not JSON
(`WS_RP_MS_ProtocolMessages__125`) leaves the reference Android Wallet on its
success screen instead of an error or a termination.

### Reproduction

14/10/2026, emulator, Wallet 2026.09.42, production
`capture-wallet.credimi.io`: the `exercise-no-redirect-response` session of
`scenarios/fcaf-wallet-solution-relying-party-verifier-response-controls.yaml`
(066) and its `exercise-non-json-response` session (125); open the deeplink,
unlock, share a PID, and observe the terminal screen.

### Observed

- After sharing, the Wallet shows `Authentication successful ... you
  successfully shared the following information` and no `Oups! Something went
  wrong` error; the guarded `no-redirect-wallet-error` / `non-json-wallet-error`
  screenshots are never stored.
- Capture cannot say whether an error arrives later, because the Wallet has
  already posted its Authorization Response.

### Expected

- 066 source: "The Wallet raises an invalid_request error or terminates
  transaction".
- 125 source: "the wallet detects that the verifier's HTTP 200 response body is
  not valid JSON and returns an error".

A success screen is neither an error nor a termination of the transaction.

### Notes

For 066 the finding is a judgment on "terminates transaction": staying on the
success screen could be argued to end the interaction, but the source's
alternative is an invalid_request error, and neither is evidenced. For 126 (an
HTTP 400 reply with a JSON error body) the Wallet does show
`Oups! ... Verifier rejected the response`, so the error path exists.

### FCAF Impact

- Tests: `WS_RP_IA_MainInteraction__066` and
  `WS_RP_MS_ProtocolMessages__125` now assert a named error screenshot
  (`no-redirect-wallet-error` / `non-json-wallet-error`) and are expected to
  FAIL on this Wallet; `WS_RP_MS_ProtocolMessages__126` passes.
