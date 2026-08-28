# WS_RP_MS_Metadata_081

## Objective
Verify that the EUDI Wallet accepts a JOSE-based Referenced Token containing a "status" claim with a valid "status_list" JSON Object.

## References
[Token Status List (TSL) draft-20] Section 6.2

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in JWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Referenced Token includes the Status element that contains the Status_list element (index and URI)
C) The EUDI Wallet has retrieved a Status List Token from the referenced URI
D) The Status List Token is validated
E)The Issuer has provided a JOSE-based Referenced Token to the EUDI Wallet.
F) The Referenced Token contains a "status" claim containing a valid "status_list" JSON Object.

## Test Scenario
1. Verify the presence and content of the "status" claim.

## Expected results
1. The "status" claim contains a valid "status_list" object and the token is accepted.

