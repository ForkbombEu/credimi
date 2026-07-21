# WS_RP_MS_Metadata_090

## Objective
Verify that the EUDI Wallet rejects a JOSE-based Referenced Token where the required "uri" claim within "status_list" is missing.

## References
[Token Status List (TSL) draft-20] Section 6.2

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in JWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Issuer has provided a JOSE-based Referenced Token to the EUDI Wallet.
C) The Referenced Token does not contain the "uri" claim within "status_list".

## Test Scenario
1. Verify the Wallet's handling of the missing "uri" claim.

## Expected results
1. The Referenced Token is rejected.

