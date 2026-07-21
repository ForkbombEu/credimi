# WS_RP_MS_Metadata_095

## Objective
Verify that the EUDI Wallet accepts a COSE-based Referenced Token where the "idx" field within "status_list" is a valid unsigned integer (major type 0)

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
F) The Referenced Token contains the "idx" field within "status_list" set to a valid unsigned integer.

## Test Scenario
1. Verify the value of the "idx" field within "status_list" is of major type 0.

## Expected results
1. The "idx" value is a valid unsigned integer and the token is accepted.

