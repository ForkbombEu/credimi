# WS_RP_IA_Metadata_011

## Objective
Test that the wallet rejects the Authorization Request if aud does NOT match the issuer claim value in the wallet metadata.

## References
[OIDF.OID4VP] section 5.8

## Profile applicability
Dynamic discovery

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The Wallet receives a Request Object (Dynamic Discovery) where the aud claim does NOT match the iss value in the wallet metadata.
3. The Wallet returns a response to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. The Wallet rejects the Request Object due to the aud mismatch and does not proceed to credential selection or user authorization.
3. The Wallet returns an error response (e.g. invalid_request).