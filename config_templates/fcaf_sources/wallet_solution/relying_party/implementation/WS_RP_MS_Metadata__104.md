# WS_RP_MS_Metadata_104

## Objective
Verify that when the client_metadata parameter is included in an Authorization Request, it is a valid UTF-8 encoded JSON object and that the Wallet processes only the defined metadata parameters.

## References
[OIDF.OID4VP] section 5.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends Authorization Request with client_metadata as a valid UTF-8 encoded JSON object containing only defined metadata parameters.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet accepts the request and processes only the defined metadata parameters from client_metadata.

