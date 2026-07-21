# WS_RP_MS_Metadata_117

## Objective
Verify that when the Wallet receives an Authorization Request using the verifier_attestation Client Identifier Prefix where the original Client Identifier does NOT match the subclaim of the Verifier attestation JWT, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using the verifier_attestation: prefix where the original Client Identifier does NOT match the subclaim of the attestation JWT.
3. Wallet parses the Authorization Request.
4. Wallet validates the Client Identifier against the subclaim and detects the mismatch.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet rejects the Authorization Request and returns an invalid_request error due to Client Identifier / subclaim mismatch; presentation flow is not initiated.

