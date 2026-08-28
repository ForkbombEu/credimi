<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# FCAF code validators and DataModel test implementation plan

## 1. Goal

Implement:

1. Every FCAF validator that can make its decision deterministically from captured evidence in Go.
2. Every YAML test definition under the relying-party `DataModel` source folder.
3. The SD-JWT VC and ISO mdoc evidence decoders required by those validators.
4. Positive and negative unit-test vectors proving that each validator rejects invalid evidence as well as accepting valid evidence.

Source test directory:

```text
../eudi-doc-functional-conformance-assessment/docs/fcaf/suts/wallet_solution/relying_party/DataModel
```

Target test directory:

```text
config_templates/fcaf/wallet_solution/relying_party/tests
```

The source currently contains 142 tests:

| Section | Tests |
| --- | ---: |
| `AddressData` | 42 |
| `CredentialMetadata` | 44 |
| `IdentifyingData` | 56 |
| **Total** | **142** |

The four `CredentialStructure_007` through `CredentialStructure_010` tests are behavioral DCQL tests. They belong in the DataModel YAML catalog, but their verdict depends on purpose-built pipelines and pipeline evidence rather than only credential-value validators.

## 2. Non-goals

This plan does not implement:

- Pipeline YAML files. They are authored separately using `FCAF_REQUIRED_PIPELINES.md`.
- Revocation, Status List Token, or trusted-list infrastructure listed in `FCAF_UNSUPPORTED_TESTS.md`.
- Browser Digital Credentials API, multi-device, or resource-exhaustion tests.
- Network lookups from validators.
- Semantic validation based only on whether a pipeline passed. A code validator must inspect the relevant raw or decoded evidence.

## 3. Required design rules

### 3.1 Evidence decoders and validators have different responsibilities

- A pipeline returns raw SD-JWT VC, raw mdoc, HTTP request/response data, session metadata, screenshots, and run metadata.
- An evidence decoder parses one raw artifact into a lossless typed representation.
- A validator receives that representation and checks one normative requirement.
- A validator must not start pipelines, call Temporal, make HTTP requests, or inspect PocketBase.
- A validator must return `error` only for bad configuration or unusable evidence. A well-formed but non-conformant value returns `fail`.

### 3.2 Preserve raw evidence

Decoded evidence must retain the original artifact:

- `SDJWTPresentation.Raw` remains the complete presented SD-JWT.
- The mdoc representation must retain the raw encoded mdoc bytes or base64/base64url input.
- Parsed values must preserve enough CBOR metadata to distinguish text strings, byte strings, unsigned integers, arrays, maps, and tagged dates.
- Reports continue to expose the real artifact as evidence. Pipeline-run URLs remain separate `pipeline.*.run` evidence.

### 3.3 Validators are reusable and narrowly scoped

Do not create one Go validator type per FCAF test. Implement reusable primitives parameterized by claim path, namespace, element, limits, allowed values, or expected format.

Examples:

```yaml
validator: sdjwt.claim_utf8_string
params:
  claim: address.locality
```

```yaml
validator: mdoc.element_country_code
params:
  namespace: eu.europa.ec.eudi.pid.1
  element: resident_country
```

### 3.4 Nested paths are explicit

SD-JWT claims such as `address.country` are nested paths, not literal top-level keys. Implement one path resolver and use it in every SD-JWT validator.

Path behavior:

- `address.country` resolves `claims["address"]["country"]`.
- Array indexes are not needed for the DataModel YAML and must not be added unless a source test requires them.
- Missing intermediate objects produce a normal validation failure.
- A literal fallback for dotted keys is allowed only if existing evidence proves such keys are emitted literally; otherwise do not add it.

### 3.5 No validator parameters come from the assessment API

All normative values are fixed in:

- Test YAML parameters when they vary by test, such as claim or element name.
- Shared precondition YAML parameters when they define the PID profile, such as mandatory claims.
- Go constants or packaged static tables when they are stable standards data, such as accepted sex values.

The API supplies only selected test IDs and `runner_id`.

## 4. Phase 0: freeze the inventory and establish traceability

1. Create a machine-readable inventory test in `pkg/fcaf/catalog` that scans the source or a checked-in manifest and records all 142 expected IDs.
2. Record, for every source test:
   - source relative path;
   - format: `ietf_sd_jwt_vc`, `iso_mdoc`, or behavioral/DCQL;
   - PID attribute identifier;
   - objective;
   - expected-result clauses;
   - normative references;
   - required pipeline evidence;
   - validator IDs;
   - prerequisite test IDs.
3. Store the maintained manifest under:

   ```text
   config_templates/fcaf_sources/wallet_solution/relying_party/implementation/datamodel-test-manifest.yaml
   ```

4. Treat the manifest as development input only. Runtime code must load only files under `config_templates/fcaf`.
5. Add a test that fails when:
   - a source DataModel ID is absent from the manifest;
   - an ID is duplicated;
   - a manifest entry has no target YAML;
   - a target YAML exists for an unknown ID.
6. Do not generate normative assertions from filename patterns. Read each Markdown source because numbered variants are not globally consistent.

Completion gate:

- Exactly 142 unique DataModel source IDs are inventoried.
- Existing three email YAMLs map to their manifest entries without changing behavior.

## 5. Phase 1: strengthen the validator contract and registry

### Step 1.1: retain the existing interface

Keep `pkg/fcaf/validators/result.go` as the single contract:

- `Validator.ID() string`
- `Validator.Validate(context.Context, Input) Result`
- statuses `pass`, `fail`, and `error`

Do not create a second validator framework.

### Step 1.2: make parameters fail closed

For every validator:

- Decode parameters with `DecodeParams`.
- Reject missing required parameters with `StatusError`.
- Reject unsupported parameter values with `StatusError`.
- Never silently choose defaults for normative limits.
- Include the claim/element path and rejected value category in failure messages.
- Do not include a complete credential in the message because it is already available through report evidence.

### Step 1.3: divide implementation files by domain

Refactor only when adding the corresponding validators:

```text
pkg/fcaf/validators/
  result.go
  registry.go
  value.go
  sdjwt.go
  mdoc.go
  pid.go
  oid4vp.go
  jose.go
  crypto.go
```

Delete a legacy validator only after all catalog references and tests prove it is unused.

### Step 1.4: registry coverage

Add every production validator to `DefaultRegistry`.

Add a registry test that:

- asserts the exact expected validator ID set;
- rejects duplicate IDs;
- invokes each validator through the registry at least once;
- scans all test and precondition YAMLs and fails if any referenced validator is not registered.

Completion gate:

- No YAML can load with an unknown validator.
- Duplicate IDs fail during worker startup or catalog construction, not during a test run.

## 6. Phase 2: implement shared value-validation primitives

Implement these internal helpers before format-specific validators:

| Helper | Required behavior |
| --- | --- |
| `resolveObjectPath` | Resolve dot-separated nested object paths with precise missing/type errors. |
| `valueTypeName` | Distinguish string, integer, number, boolean, object, array, bytes, and null. |
| `validateUTF8String` | Require Go string and valid UTF-8. |
| `validateStringLength` | Count Unicode code points, not bytes. Support explicit inclusive minimum and maximum. |
| `validateDateString` | Validate exact `YYYY-MM-DD`, real Gregorian dates, leap years, and year range required by the source test. |
| `validateDateTimeString` | Validate the exact RFC 3339/profile form stated by the source test. |
| `validateCountryCode` | Accept uppercase ISO 3166-1 alpha-2 plus user-assigned `AA`, `QM`-`QZ`, `XA`-`XZ`, and `ZZ`. |
| `validateInternationalPhone` | Require `+`, country code and digits only, with at least eight Unicode characters as required by the tests. |
| `validateStringArray` | Require an array, minimum item count, and string items. |
| `validateAllowedInteger` | Require an integer, not a fractional JSON number, and compare against an explicit allowed set. |
| `validateJPEG` | Decode the supplied byte representation and require JPEG SOI bytes `FF D8`; reject malformed data URLs/base64. |

Implementation requirements:

1. Use table-driven unit tests for boundaries and malformed values.
2. Add positive and negative fixtures only where static files improve readability, such as invalid UTF-8 and encoded images.
3. Do not use a network-backed country-code package. Use a reviewed static table or an existing repository dependency.
4. If a new dependency is needed, first inspect `eudi-conformance-evidence` and existing Credimi dependencies. Add a new module only when existing code cannot provide standards-compliant parsing.
5. Ensure JSON numbers are checked for integral value before treating them as integer claims.

Completion gate:

- Every helper has positive, negative, wrong-type, missing-value, and boundary tests.

## 7. Phase 3: complete SD-JWT VC evidence decoding

Target files:

```text
pkg/fcaf/evidence/extract.go
pkg/fcaf/evidence/bundle.go
pkg/fcaf/evidence/lookup.go
pkg/fcaf/evidence/lookup_test.go
```

### Step 3.1: make disclosure reconstruction complete

Verify and test that `sdjwt.presentation` and `sdjwt.vp_token_json`:

- split issuer JWT, disclosures, and optional key-binding JWT correctly;
- decode protected headers and issuer payload;
- calculate each disclosure digest using `_sd_alg`;
- match disclosures to `_sd` entries;
- recursively reconstruct nested disclosed objects;
- reconstruct array disclosures when present;
- reject malformed base64url, malformed disclosure arrays, duplicate conflicting disclosures, and unsupported digest algorithms;
- preserve undisclosed digest information separately from disclosed claims;
- retain the raw presentation.

Parsing a disclosure without verifying its digest is insufficient.

### Step 3.2: expose a stable typed value

Keep or extend `evidence.SDJWTPresentation` with:

- `Raw`;
- `ProtectedHeaders`;
- `IssuerPayload`;
- `Claims`;
- `KeyBinding`;
- digest/disclosure verification metadata.

Do not make validators parse compact JWT segments repeatedly.

### Step 3.3: fix nested claim lookup

Update the common SD-JWT claim resolver so every validator can use:

- `email`;
- `address.formatted`;
- `address.locality`;
- `address.country`;
- `address.house_number`;
- `address.postal_code`;
- `address.region`;
- `address.street_address`;
- nested `place_of_birth` members.

Completion gate:

- Tests demonstrate valid and invalid nested disclosures.
- Existing email tests still pass.
- Invalid disclosure hashes cannot produce passing claim validators.

## 8. Phase 4: implement SD-JWT VC validators

### Step 4.1: retain and generalize current validators

Retain these IDs where their behavior is correct:

- `sdjwt.claim_present`
- `sdjwt.claim_type`
- `sdjwt.claim_string_prefix`
- `sdjwt.claim_utf8_string`
- `sdjwt.claim_rfc5322_email`
- `pid.sdjwt_vct_pid`
- `pid.sdjwt_required_mandatory_claims_present`

Change internals to use the shared nested-path resolver.

### Step 4.2: add DataModel validators

Implement:

| Validator ID | Parameters | Checks |
| --- | --- | --- |
| `sdjwt.claim_non_empty_utf8_string` | `claim` | String, valid UTF-8, at least one code point. |
| `sdjwt.claim_international_phone` | `claim`, `min_length` | International `+` format and digits. |
| `sdjwt.claim_country_code` | `claim` | ISO/user-assigned alpha-2 code. |
| `sdjwt.claim_date_format` | `claim` | Exact `YYYY-MM-DD` syntax. |
| `sdjwt.claim_valid_date` | `claim` | Real calendar date. |
| `sdjwt.claim_rfc3339_datetime` | `claim` | Required RFC 3339/profile date-time. |
| `sdjwt.claim_string_array` | `claim`, `min_items` | Non-empty array of strings. |
| `sdjwt.claim_country_code_array` | `claim`, `min_items` | Every entry is an accepted country code. |
| `sdjwt.claim_object` | `claim` | JSON object. |
| `sdjwt.claim_object_keys` | `claim`, `allowed`, `min_properties`, `max_properties` | Only allowed unique object members and required cardinality. |
| `sdjwt.claim_object_string_values` | `claim`, `keys` | Present members are UTF-8 strings. |
| `sdjwt.claim_nested_string_max_length` | `claim`, `member`, `max_length` | Nested UTF-8 value and code-point limit. |
| `sdjwt.claim_integer_allowed` | `claim`, `allowed` | Integral numeric claim in the allowed set. |
| `sdjwt.claim_jpeg_data_url` | `claim` | `data:image/jpeg;base64,...`, valid base64, JPEG SOI. |

Use `sdjwt.claim_type` for simple string/number/object checks only when it exactly covers the source requirement. Do not combine presence with shape when the source defines separate tests; preserve test dependencies.

### Step 4.3: add non-DataModel SD-JWT/crypto validators

Implement code-only validators identified by the full relying-party inventory when raw evidence is sufficient:

- SD-JWT disclosure digest algorithm and digest verification;
- key-binding JWT structure and required bindings;
- issuer JWT JOSE header requirements;
- `x5c` certificate-chain structure, signature verification, and trust-anchor omission checks;
- holder-binding confirmation key structure;
- claim-selective-disclosure checks comparing request and presentation.

Keep acceptance/rejection behavior in pipelines. The code validator checks the captured artifact; the test YAML may require both.

Completion gate:

- Every SD-JWT validator has direct unit tests with valid input, invalid value, wrong type, missing claim, and malformed presentation.

## 9. Phase 5: implement raw ISO mdoc decoding

The current `mdoc.namespace_element_present` validator operates on an assumed map and does not prove CBOR major types, tags, issuer-signed item structure, or error items. Replace that assumption with a typed decoder.

### Step 5.1: choose the parser deliberately

Before adding code:

1. Inspect `github.com/forkbombeu/eudi-conformance-evidence` for an mdoc/CBOR parser already used by the Forkbomb ecosystem.
2. Inspect existing transitive dependencies for standards-compliant CBOR and COSE support.
3. If neither is sufficient, propose the smallest maintained CBOR/COSE dependency and record the decision before changing `go.mod`.
4. Do not write a custom CBOR parser.

### Step 5.2: define a typed mdoc presentation

Add a representation under `pkg/fcaf/evidence` that retains:

- raw mdoc bytes;
- document `docType`;
- issuer namespaces;
- each `IssuerSignedItem` and its original encoded bytes;
- element identifier;
- decoded value;
- CBOR major type;
- CBOR tag, including full-date tag 1004 where applicable;
- document and namespace errors, including `ErrorItem`;
- issuer authentication/MSO data needed by later cryptographic validators;
- device-signed/device-authentication data needed by later validators.

### Step 5.3: add decoder IDs

Add explicit output decoders, for example:

```yaml
decoder: mdoc.presentation
```

If the verifier wraps the raw mdoc in a `vp_token` JSON object, add one wrapper decoder analogous to `sdjwt.vp_token_json`. The wrapper decoder extracts the raw mdoc and delegates to the same mdoc parser.

Do not add one decoder per pipeline.

### Step 5.4: decoder tests

Use real positive and negative fixtures:

- one PID mdoc with text, integer, array, map, full-date, and JPEG elements;
- missing namespace;
- missing element;
- `ErrorItem` for a requested element;
- wrong CBOR major type;
- wrong/missing tag 1004;
- malformed CBOR;
- wrong `docType`.

Completion gate:

- Validators receive typed mdoc evidence, never an unverified arbitrary map.
- Raw bytes remain available in the report evidence.

## 10. Phase 6: implement ISO mdoc validators

Implement:

| Validator ID | Parameters | Checks |
| --- | --- | --- |
| `mdoc.doc_type` | `expected` | Presented document has PID `docType`. |
| `mdoc.namespace_element_present` | `namespace`, `element` | Exactly one requested element exists and no corresponding `ErrorItem` exists. |
| `mdoc.element_cbor_type` | `namespace`, `element`, `major_type` | Original value has required CBOR major type. |
| `mdoc.element_utf8_string` | `namespace`, `element`, optional limits | CBOR text string, valid UTF-8, code-point bounds. |
| `mdoc.element_full_date` | `namespace`, `element` | Tag 1004, text-string payload, exact date syntax, real date. |
| `mdoc.element_rfc3339_datetime` | `namespace`, `element` | Required date-time tag/type and valid RFC 3339/profile value. |
| `mdoc.element_country_code` | `namespace`, `element` | Accepted alpha-2/user-assigned country code. |
| `mdoc.element_string_array` | `namespace`, `element`, `min_items` | CBOR array with required item count and text-string items. |
| `mdoc.element_country_code_array` | `namespace`, `element`, `min_items` | Every item is an accepted code. |
| `mdoc.element_map_shape` | `namespace`, `element`, `allowed_keys`, bounds | CBOR map, allowed unique keys, required cardinality. |
| `mdoc.element_map_text_values` | `namespace`, `element`, `keys` | Selected map values are CBOR text strings. |
| `mdoc.element_map_member_country_code` | namespace/element/member | Nested country code. |
| `mdoc.element_map_member_utf8_max_length` | namespace/element/member/limit | Nested UTF-8 string and code-point maximum. |
| `mdoc.element_unsigned_integer_allowed` | namespace/element/allowed | CBOR major type 0 and allowed integer value. |
| `mdoc.element_jpeg` | `namespace`, `element` | CBOR byte string and JPEG SOI bytes. |
| `mdoc.no_error_item` | `namespace`, `element` | Explicit reusable error-item assertion when a test needs it independently. |

Add non-DataModel mdoc/crypto validators when supported by raw evidence:

- issuer authentication COSE structure and signature;
- MSO digest algorithm;
- issuer namespace value digests;
- device authentication/signature;
- session transcript binding;
- validity information and certificate-chain structure.

Do not implement revocation or trusted-list verdicts until the infrastructure in `FCAF_UNSUPPORTED_TESTS.md` exists.

Completion gate:

- Each validator proves both decoded semantic value and required original CBOR representation.

## 11. Phase 7: implement protocol, JOSE, DCQL, and crypto code validators

This phase covers the remaining full-suite validators that can inspect captured artifacts directly. It does not replace pipelines that prove wallet behavior.

### 11.1 OID4VP response validators

Implement or complete:

- authorization response contains or omits `vp_token`;
- response URI POST evidence;
- nonce and state binding with required-presence and character checks;
- device-binding proof fields and cryptographic binding;
- verifier-info integrity;
- request URI/request object structure;
- client metadata constraints.

The existing `oid4vp.device_binding` and `oid4vp.nonce_state_binding` implementations are placeholders because they can pass without proving required fields. Rewrite them with exact input schemas and negative tests.

### 11.2 JOSE/JWE validators

Rewrite `jose.jwe_encrypted_response` so five compact segments alone are not considered valid. Validate:

- compact or JSON serialization syntax required by the test;
- protected header;
- required `alg` and `enc`;
- recipient key selection;
- authenticated decryption when private-key evidence is available;
- required claims in the decrypted response.

### 11.3 DCQL validators

Implement:

- claim-path-pointer evaluation against the presented credential;
- `credential_sets` constraints;
- matching of each `credentials` entry;
- no-match response;
- `claim_sets` selection.

For each DCQL validator, input must contain both:

- the captured request/DCQL query;
- the wallet response or decoded presented credentials.

Pipeline success alone cannot prove that constraints were applied.

### 11.4 Cryptographic validators

Implement only when all required raw inputs are available:

- SD-JWT disclosure SHA-256 checks;
- mdoc digest SHA-256 checks;
- request hash algorithm policy inspection;
- response encryption algorithm inspection;
- certificate-chain and signature checks.

Where the test expects the wallet to reject an unsupported algorithm, use:

- pipeline outcome/UI or verifier response to prove rejection;
- code validator to prove the request actually contained the intended unsupported algorithm.

Completion gate:

- Each code validator identifies its exact required evidence fields.
- No validator returns pass merely because input is non-null.

## 12. Phase 8: create shared PID preconditions for mdoc

Add preconditions parallel to the SD-JWT preconditions:

```text
pipeline.pid.presentation.mdoc.all-ics-elements
assertion.pid.presentation.mdoc.doc-type-pid
assertion.pid.presentation.mdoc.required-mandatory-elements-presented
```

Pipeline precondition requirements:

- `pipeline_id` points to the manually authored PID mdoc presentation pipeline.
- Output path extracts the raw mdoc only.
- Decoder parses it as an mdoc presentation.
- `runner_id` remains the only caller-controlled runtime pipeline value.

Assertion precondition requirements:

- PID `docType` is checked from decoded evidence.
- Mandatory PID mdoc element names are fixed in the precondition YAML.
- Mandatory-element presence checks the PID namespace.
- Do not add optional/ICS attributes such as email unless the PID rulebook marks them mandatory.

Completion gate:

- One real mdoc pipeline output can execute the common preconditions and a sample mdoc DataModel test.

## 13. Phase 9: write all DataModel YAML definitions

### 13.1 YAML shape

Every file must use only the runtime DSL fields:

```yaml
id:
title:
source:
suite:
applicability:
normative_references:
preconditions:
evidence:
assertions:
verdict:
```

Do not add summaries, drafts, implementation notes, pipeline output paths, or unused metadata.

Rules:

- `source.path` is relative to the relying-party source root.
- Normative references are copied from the source Markdown and include links.
- Pipeline JSON paths exist only in precondition YAML.
- Test evidence binds to a named precondition output.
- Each assertion checks one expected-result requirement where practical.
- `verdict.pass_when` is always `all_assertions_pass`.
- A test dependency is added only when the source precondition explicitly requires the earlier test result.

### 13.2 SD-JWT test template

```yaml
preconditions:
  - ref: pipeline.pid.presentation.sdjwt.all-ics-claims
  - ref: assertion.pid.presentation.sdjwt.vct-pid
  - ref: assertion.pid.presentation.sdjwt.required-mandatory-claims-presented
evidence:
  pid_sdjwt:
    from: pipeline.pid.presentation.sdjwt.all-ics-claims.outputs.pid_sdjwt
assertions:
  - id: <requirement-specific-id>
    validator: <registered-validator-id>
    input: evidence.pid_sdjwt
    params:
      claim: <claim-path>
verdict:
  pass_when: all_assertions_pass
```

### 13.3 mdoc test template

```yaml
preconditions:
  - ref: pipeline.pid.presentation.mdoc.all-ics-elements
  - ref: assertion.pid.presentation.mdoc.doc-type-pid
  - ref: assertion.pid.presentation.mdoc.required-mandatory-elements-presented
evidence:
  pid_mdoc:
    from: pipeline.pid.presentation.mdoc.all-ics-elements.outputs.pid_mdoc
assertions:
  - id: <requirement-specific-id>
    validator: <registered-validator-id>
    input: evidence.pid_mdoc
    params:
      namespace: eu.europa.ec.eudi.pid.1
      element: <element-name>
verdict:
  pass_when: all_assertions_pass
```

### 13.4 AddressData mapping

Write all 42 files using these mappings:

| Data identifier | SD-JWT claim | mdoc element | Additional shape |
| --- | --- | --- | --- |
| email address | `email` | `email_address` | UTF-8 and RFC 5322 |
| mobile phone | `phone_number` | `mobile_phone_number` | international phone |
| resident address | `address.formatted` | `resident_address` | UTF-8 |
| resident city | `address.locality` | `resident_city` | UTF-8 |
| resident country | `address.country` | `resident_country` | country code |
| house number | `address.house_number` | `resident_house_number` | UTF-8 |
| postal code | `address.postal_code` | `resident_postal_code` | UTF-8 |
| state | `address.region` | `resident_state` | UTF-8 |
| street | `address.street_address` | `resident_street` | UTF-8 |

For each family:

1. Presence variant checks presence.
2. Encoding/type variant depends on presence when required by the source.
3. Semantic variant depends on presence and checks the exact format.
4. mdoc presence also proves no `ErrorItem`.

### 13.5 IdentifyingData mapping

Write all 56 files:

| Data identifier | SD-JWT claim | mdoc element | Validators |
| --- | --- | --- | --- |
| birth date | `birthdate` | `birth_date` | presence, string/text, exact format, valid date; mdoc tag 1004 |
| birth place | `place_of_birth` | `place_of_birth` | object/map shape, allowed members, text values, country, region/locality lengths |
| family name | `family_name` | `family_name` | presence and UTF-8 |
| family name at birth | `birth_family_name` | `family_name_birth` | presence and UTF-8 |
| given name | `given_name` | `given_name` | presence and UTF-8 |
| given name at birth | `birth_given_name` | `given_name_birth` | presence and UTF-8 |
| nationality | `nationalities` | `nationality` | non-empty string array and country code per item |
| personal administrative number | `personal_administrative_number` | same | presence and UTF-8 |
| portrait | `picture` | `portrait` | string/data URL/JPEG or CBOR bytes/JPEG |
| sex | `sex` | `sex` | number/unsigned integer and allowed values `0`-`6`, `9` |

Read each birth-place source separately. Do not copy the apparent source typo between region/locality tests into validator parameters.

### 13.6 CredentialMetadata mapping

Write all 44 files:

| Family | SD-JWT claim | mdoc element | Validators |
| --- | --- | --- | --- |
| document number | `document_number` | `document_number` | presence and UTF-8 |
| domestic indicator/details | source-defined claim | source-defined element | exact type/shape from Markdown |
| expiry date | `date_of_expiry` | `expiry_date` | presence, representation, exact date, valid date |
| issuance date | `date_of_issuance` | `issuance_date` | presence, representation, date/date-time rules from each source |
| issuing authority | `issuing_authority` | `issuing_authority` | presence and UTF-8 |
| issuing country | `issuing_country` | `issuing_country` | presence, UTF-8, country code |
| issuing jurisdiction | `issuing_jurisdiction` | `issuing_jurisdiction` | presence, UTF-8, source-defined jurisdiction shape |

Do not infer claim names for `Domestic`, issuance, or jurisdiction variants from this table alone. Confirm each source Markdown and the PID rulebook mapping in the manifest before writing YAML.

### 13.7 CredentialStructure behavioral YAMLs

For `CredentialStructure_007`–`010`:

1. Bind pipeline evidence containing the exact DCQL request and wallet response.
2. Use pipeline preconditions representing the correct positive or negative request fixture.
3. Invoke DCQL code validators over a combined request/response evidence object.
4. For no-match behavior, accept only the outcomes allowed by the source: invalid-request error or empty VP token.
5. Do not use the normal “all PID claims” presentation pipeline because it cannot prove the requested DCQL constraint.

## 14. Phase 10: fixtures and validator tests

Organize fixtures by evidence type:

```text
fixtures/fcaf/
  validators/
    sdjwt/
    mdoc/
    oid4vp/
    dcql/
    jose/
```

Fixture rules:

- Fixtures are immutable test inputs.
- Include the normative case in the filename.
- Pair positive and negative vectors.
- Keep secrets and real personal data out of fixtures.
- Use synthetic names, dates, identifiers, certificates, and images.
- Generate cryptographic fixtures with a documented test helper; do not hand-edit signatures.

Required boundary vectors include:

- malformed and valid UTF-8;
- empty and one-character strings;
- seven- and eight-character phone values;
- leap and non-leap dates;
- invalid month/day combinations;
- ISO, user-assigned, lowercase, malformed, and unknown country codes;
- empty, mixed-type, and valid nationality arrays;
- integer, fractional, string, and out-of-range sex values;
- valid and invalid JPEG data URLs and CBOR byte strings;
- valid and malformed nested birth-place objects/maps;
- valid and invalid SD-JWT disclosure digests;
- valid and malformed mdoc CBOR tags and major types.

## 15. Phase 11: catalog and DSL validation

Add catalog-level validation that runs after parsing:

1. Every precondition reference resolves.
2. Every test dependency resolves.
3. Every evidence binding resolves to a declared pipeline output.
4. Every validator exists in `DefaultRegistry`.
5. No dependency cycle exists.
6. Test IDs equal filenames.
7. Source paths are unique and correspond to the expected source manifest.
8. All 142 DataModel YAML files load.
9. No unexpected Markdown file exists in the runtime template directory.
10. Every test has at least one normative reference and assertion.

Add a single test that loads the complete production catalog, builds the execution graph for all DataModel tests, and verifies graph expansion succeeds.

## 16. Phase 12: end-to-end rollout

Roll out in small executable slices:

### Slice A: finish SD-JWT AddressData

- Implement nested claim lookup and generic string/country/phone validators.
- Add the remaining SD-JWT AddressData YAMLs.
- Execute them against the existing all-claims SD-JWT pipeline.

### Slice B: SD-JWT IdentifyingData

- Add date, object, array, country, image, and integer validators.
- Add all SD-JWT IdentifyingData YAMLs.
- Execute the complete slice.

### Slice C: SD-JWT CredentialMetadata

- Add date-time and jurisdiction checks.
- Add all non-DCQL SD-JWT metadata YAMLs.
- Execute the complete slice.

### Slice D: mdoc foundation and AddressData

- Implement the real mdoc decoder and shared mdoc preconditions.
- Add all mdoc AddressData YAMLs.
- Execute against a real mdoc pipeline output.

### Slice E: mdoc IdentifyingData and CredentialMetadata

- Add CBOR tag, map, array, integer, and JPEG validators.
- Add the remaining mdoc YAMLs.
- Execute the complete slice.

### Slice F: CredentialStructure

- Add the four dedicated DCQL pipelines/preconditions.
- Add combined request/response evidence extraction.
- Execute tests `007`–`010`.

Do not generate all YAMLs first and postpone execution. Each slice is complete only after a real Temporal assessment produces understandable pass and failure reports.

## 17. Validation commands

During each slice:

```bash
go test -tags=unit ./pkg/fcaf/... -count=1
go test -tags=unit ./pkg/workflowengine/activities ./pkg/workflowengine/workflows -run FCAF -count=1
```

Before declaring the implementation complete:

```bash
make fmt
go mod tidy -diff
go mod verify
make lint
make test
```

Also run the repository revive check used by CI and account for `govulncheck` if the local aggregate cannot run it.

Run at least these real API assessments:

1. One SD-JWT address family with presence, type, and semantic tests.
2. One nested SD-JWT address claim.
3. One SD-JWT date, nationality array, portrait, and sex family.
4. One mdoc text, full-date, map, array, JPEG, and unsigned-integer family.
5. All four DCQL CredentialStructure tests.
6. One intentionally invalid fixture per validator family to verify the workflow fails and reports the exact assertion reason.

## 18. Definition of done

The work is complete only when:

- All code-based validators in the relying-party inventory are implemented or explicitly moved to `FCAF_UNSUPPORTED_TESTS.md` with a concrete missing-evidence reason.
- All 142 DataModel source tests have runtime YAML definitions.
- All YAML validator references resolve.
- SD-JWT disclosures are cryptographically matched to digests before claims are validated.
- mdoc tests inspect a typed, lossless CBOR representation.
- Every validator has positive and negative tests.
- All DataModel definitions load into one acyclic execution graph.
- Pipeline evidence is reused across dependent tests.
- Reports contain the real SD-JWT/mdoc/request/response evidence used by assertions.
- Pipeline preconditions reference separate pipeline-run URL evidence.
- A failed precondition skips its dependent assertion and explains why.
- A failed assertion fails the assessment workflow and identifies the test, assertion, evidence key, and reason.
- The required repository validation passes, or exact environmental blockers are documented.

## 19. Implementation checklist for each new validator

Use this checklist for every validator:

1. Locate every source test that needs it.
2. Write the exact normative rule in the validator test name/table.
3. Define the input evidence type.
4. Define required YAML parameters.
5. Add wrong-configuration tests returning `error`.
6. Add missing-input and wrong-type tests returning `fail`.
7. Add minimum/maximum boundary tests.
8. Implement the validator without I/O.
9. Register it in `DefaultRegistry`.
10. Add one production YAML reference.
11. Run focused tests.
12. Run complete catalog loading.
13. Execute one real passing assessment.
14. Execute one real failing assessment.
15. Verify report evidence is the artifact checked by the validator.

## 20. Implementation checklist for each DataModel YAML

1. Open the source Markdown.
2. Copy the exact test ID.
3. Copy objective into a concise title without changing meaning.
4. Record every normative reference and section.
5. Determine SD-JWT, mdoc, or behavioral applicability.
6. Map each source precondition to a pipeline, assertion, or test dependency.
7. Bind only evidence needed by this test.
8. Map every expected-result clause to an assertion.
9. Use a registered validator and explicit parameters.
10. Keep output JSON paths in pipeline precondition YAML only.
11. Parse and validate the YAML.
12. Verify dependencies are acyclic.
13. Add the YAML to the 142-test coverage check.
14. Run it with valid evidence.
15. Run its validator with invalid evidence and verify the reason is specific.
