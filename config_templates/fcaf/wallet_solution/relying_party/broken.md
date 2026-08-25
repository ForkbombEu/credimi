<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# FCAF relying-party verifier backlog

Audit date: 25/08/2026

This file records capabilities missing from the verifier or capture service. It
is not a list of Wallet failures and it is not a list of every test that still
needs implementation.

## Current classification

| State | Definitions | Executed by the catalog |
| --- | ---: | --- |
| Runnable | 209 | Yes |
| Pending implementation | 298 | No |
| Verifier blocked | 52 | No |
| Total | 559 | |

Runnable definitions live directly in `tests/`. Pending definitions live in
`tests/_implementation/pending/`. The definitions listed below live in
`tests/_implementation/verifier-blocked/`. The catalog loader intentionally
skips `_implementation`.

Moving a definition into the runnable directory requires a dedicated scenario,
evidence bindings that resolve to real pipeline output, and assertions that
independently prove the request preconditions and the expected Wallet result.
A category-wide happy-path pipeline is not sufficient.

## Missing verifier capabilities

### Plain Authorization Request delivery

Affected test: `WS_RP_MS_ProtocolMessages__002`.

Missing behavior: construct the plain, unsigned Authorization Request required
by the source, with the authorization parameters directly in the launch URL.
The verifier must not silently replace it with a signed Request Object or a
`request_uri` by-reference flow.

Required evidence: the exact Wallet launch URL and its decoded query
parameters, plus the Wallet response and mandatory interaction evidence.

Activation criterion: a dedicated pipeline proves that the request contains
the required direct parameters and contains neither `request` nor
`request_uri`, then proves the source's expected Wallet outcome.

### Arbitrary and malformed signed Authorization Requests

Affected tests:

`WS_RP_MS_ProtocolMessages__096–098`, `100`, `108`, `110–115`, and `120–124`.

Missing behavior: sign and deliver request payloads that are not representable
by the public verifier's typed request model. This includes malformed DCQL
members, unsupported JSON types and unknown fields that must survive signing.
The service must not validate, normalize, or strip the property before the
Wallet receives it.

Required evidence: the compact signed Request Object, its decoded payload, the
exact launch URL, and the Wallet's protocol response or a content-specific UI
discontinuation result. Synthetic `json-parse` values are not Wallet evidence.

Activation criterion: each case has a dedicated malformed request fixture; a
precondition assertion proves the malformed property is present in the signed
bytes; and a separate assertion proves the exact outcome allowed by the FCAF
source.

### Configurable Wallet response endpoint

Affected tests: `WS_RP_MS_ProtocolMessages__125–128`.

Missing behavior: after receiving the Wallet submission, return the exact HTTP
status, content type, JSON or non-JSON body, and unknown response parameters
required by each scenario.

Required evidence: captured request and response status line, headers and raw
body for both directions, linked to the same verifier session.

Activation criterion: the endpoint behavior is selected per pipeline request,
the raw exchange is retained, and assertions prove both the endpoint behavior
and the Wallet's subsequent redirect or error handling.

### Raw JWE and Wallet-facing HTTP capture

Affected tests: `WS_RP_MS_ProtocolMessages__129–134`.

Missing behavior: retain the compact Authorization Response JWE before
decryption and capture the raw HTTP request sent to the Wallet-facing
`response_uri`.

Required evidence: compact JWE, protected header, encrypted payload bytes,
HTTP method, exact `Content-Type`, and the unmodified form body. A decrypted
result object cannot prove these properties.

Activation criterion: validators consume raw capture artifacts and prove
`kid`, explicit or default `enc`, payload placement, HTTP method, content type,
and exact form fields without reconstructing them from decoded output.

### Scope, client metadata, transaction data and session state

Affected tests:

`WS_RP_MS_ProtocolMessages__135–146` and `153–159`.

Missing behavior: create signed requests with controlled supported,
unsupported and malformed `transaction_data`; unknown, malformed or empty
scopes; conflicting query instructions; controlled client identifier forms;
and conflicting local or trusted-registry verifier metadata. The verifier must
also capture exact Wallet error responses and session termination.

Required evidence: raw signed request, decoded request payload, configured
local/registry state, Wallet response, and before/after session state. The
suite also needs a documented supported transaction-data schema for cases that
mutate a valid baseline.

Activation criterion: every case starts from an explicit valid baseline,
changes only the property named by the source, and proves the exact error and
state transition separately.

### Unsupported credential format delivery

Affected test: `WS_RP_MS_ProtocolMessages__150`.

Missing behavior: sign and deliver a request containing `format: vc+sd-jwt`.
The public verifier currently rejects that format while creating the request,
so the Wallet never receives the scenario.

Required evidence: signed request bytes proving the exact format value and the
Wallet's submitted `vp_formats_not_supported` response.

Activation criterion: request creation succeeds without rewriting the format,
the Wallet receives it, and the exact protocol error is captured.

### Verifier Info attestation generation

Affected tests: `WS_RP_SM_DeviceBinding__002–006`.

Missing behavior: generate `verifier_info` inputs for valid, malformed,
invalid-proof and unknown-type attestations, with controllable signed bytes and
trust material.

Required evidence: the exact `verifier_info` value sent to the Wallet, decoded
attestation fields, proof verification inputs, trust configuration, and the
Wallet response or required UI decision.

Activation criterion: each attestation variant is generated from deterministic
fixtures, its cryptographic or structural precondition is independently
validated, and the Wallet outcome is asserted separately.

## Resolved by this remediation

- Removed 376 category-wide placeholder scenarios from the runnable catalog.
- Restored the 48 definitions modified by `794e7db0` from their last
  test-specific form, then classified 32 as runnable, five as pending and 11 as
  verifier blocked.
- Classified the 336 definitions added by `794e7db0` instead of treating their
  generated category pipelines as semantic implementations.
- Made runnable pipeline filenames match the basename derived from
  `pipeline_id`, which is the lookup contract used by the FCAF CLI.
- Isolated pending and verifier-only batches, preconditions and pipelines under
  `_implementation`.
- Removed six committed SQLite/WAL runtime artifacts from `pipelines/`.
- Rechecked the 20 runnable cases with source-sensitive UI evidence. Their
  Maestro flows retain mandatory consent, successful-share, denial or invalid
  request gates as applicable; protocol and cryptographic assertions remain
  independent of screenshot existence.

## Reactivation checklist

Before moving any blocked definition back to `tests/`:

1. Produce the exact source scenario without verifier-side normalization.
2. Bind assertions to captured request, response and UI evidence from that run.
3. Prove the precondition independently from the expected result.
4. Reject opposite or nearby scenarios in validator unit tests.
5. Run the pipeline against the reference Wallet and retain the raw evidence.
6. Update `pkg/fcaf/implementation-inventory.csv` and this backlog.
