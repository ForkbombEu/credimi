# WS_RP_MS_Metadata_101

## Objective
Verify that the EUDI Wallet accepts a COSE-based Referenced Token containing the status claim at CBOR label 65535 with a valid Status CBOR structure.

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
F) The Referenced Token contains the status claim at CBOR label 65535 containing a valid Status CBOR structure.

## Test Scenario
1. Verify the presence of the status claim at CBOR label 65535.

## Expected results
1. The status claim at label 65535 contains a valid Status CBOR structure and the token is accepted.

