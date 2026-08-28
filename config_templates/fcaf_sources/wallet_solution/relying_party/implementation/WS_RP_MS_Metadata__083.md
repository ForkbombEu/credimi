# WS_RP_MS_Metadata_083

## Objective
Verify that the EUDI Wallet accepts a JOSE-based Referenced Token containing a "status_list" JSON Object within the "status" claim, with valid "idx" and "uri" fields.

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
E) The Issuer has provided a JOSE-based Referenced Token to the EUDI Wallet.
F) The Referenced Token contains a "status_list" JSON Object within the "status" claim, containing valid "idx" and "uri" fields.

## Test Scenario
1. Verify the presence of "status_list" within the "status" claim and its "idx" and "uri" fields.

## Expected results
1. The "status_list" object contains valid "idx" and "uri" fields and the token is accepted.

