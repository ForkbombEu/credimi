# FCAF pending assertions design

## Goal

Replace the eight relying-party definitions in `## Pending` with source-specific
scenarios and assertions that prove the requested claim-path or certificate
property from the same Capture Wallet session.

## Findings that drive the design

The seven TextualEncoding definitions currently share
`pipeline.dcql.encoding`, request only SD-JWT `given_name`, and invoke
`sdjwt.claim_utf8_string` without its required `claim` parameter. They cannot
prove the source-specific array traversal, mdoc path shape, error outcome, or
CBOR element described by the corresponding source tests.

`WS_RP_SM_IssuerIntegrity__014` currently proves only that its DCQL exchange
matches the generic constraint. The source instead requires an SD-JWT VC whose
issuer `x5c` chain contains the issuer signing certificate and intermediates
but excludes the trust-anchor certificate.

## Evidence rules

- Decode the signed Authorization Request captured by the session before
  asserting a supplied DCQL path; session-creation input alone is insufficient.
- For positive JSON-path cases, prove the exact requested path and the retained
  disclosure values. For negative mdoc-path cases, prove the malformed or
  absent path, no `vp_token`, and the source-required captured Wallet error.
- For positive mdoc cases, parse the returned device response and inspect the
  named namespace/element as CBOR text; a generic successful presentation does
  not prove the selected element.
- Screenshots remain visual evidence only. They never substitute for a
  captured response, error, or JOSE header.
- The issuer-integrity validator must compare complete decoded DER certificates
  against an explicit, independently obtained trust-anchor fingerprint. It
  must not infer the trust anchor merely from chain position or self-signing.

## Fixture qualification gate

The source examples require JSON `degrees` arrays for cases 008 and 011,
`org.iso.18013.5.1.first_name` for cases 017 and 021, and an independently
known issuer trust anchor for case 014. Before changing a test definition,
run a Capture session and inspect the returned artifacts to establish each
fixture. If the reference Wallet cannot hold the exact fixture, or Capture
cannot expose the named trust anchor, move only that test to `Blocked` with
the observed missing fixture/evidence; do not reuse `given_name` or a generic
success as a substitute.

## Validation

Use table-driven Go tests to cover every new validator's success and
false-positive rejection paths. Regenerate with `make fcaf-generate`, run
`go test ./pkg/fcaf/catalog ./cmd/fcaf-pipeline-gen ./pkg/fcaf/validators -count=1`,
and run `git diff --check`. A device run is additionally required for each
changed scenario before recording an implementation as reference-Wallet
verified.
