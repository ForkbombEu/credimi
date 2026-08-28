# WS_RP_MS_Metadata_086

## Objective
Verify that the EUDI Wallet rejects a JOSE-based Referenced Token where the "idx" claim within "status_list" is a negative integer.

## References
[Token Status List (TSL) draft-20] Section 6.2

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in JWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Referenced Token includes the Status element that contains the Status_list element (index and URI)
C) The Issuer has provided a JOSE-based Referenced Token to the EUDI Wallet.
D) The Referenced Token contains the "idx" claim within "status_list" set to a negative integer.

## Test Scenario
1. Verify the Wallet's handling of the negative "idx" value.

## Expected results
1. The Referenced Token is rejected.

