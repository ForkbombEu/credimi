# WS_RP_MS_Metadata_093

## Objective
Verify that the EUDI Wallet accepts a COSE-based Referenced Token containing a "status_list" CBOR structure within the Status Map, with valid "idx" and "uri" fields.

## References
[Token Status List (TSL) draft-20] Section 6.3

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Referenced Token includes the Status element that contains the Status_list element (index and URI)
C) The EUDI Wallet has retrieved a Status List Token from the referenced URI
D) The Status List Token is validated
E) The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
F) The Referenced Token contains a "status_list" CBOR structure within the Status Map, containing valid "idx" and "uri" fields.

## Test Scenario
1. Verify the presence of "status_list" within the Status Map and its "idx" and "uri" fields.

## Expected results
1. The "status_list" structure contains valid "idx" and "uri" fields and the token is accepted.

