# WS_RP_IA_Metadata_013

## Objective
Verify that when the Wallet receives a Request Object using Static Discovery metadata with an aud claim whose value is NOT the SIOPv2 static identifier "https://self-issued.me/v2", it rejects the request and returns an invalid_request error.

## References
[OIDF.OID4VP] section 5.8

## Profile applicability
static discovery

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using Static Discovery metadata with the aud claim set to a value OTHER than "https://self-issued.me/v2".
3. Wallet parses the Request Object.
4. Wallet validates the aud claim against the expected SIOPv2 static identifier.
5. Wallet detects the mismatch.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet validates the aud claim and detects that it does not equal "https://self-issued.me/v2".
5. Wallet rejects the request and returns an invalid_request error due to the invalid aud value.

