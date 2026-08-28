# WS_RP_MS_Metadata_096

## Objective
Verify that the EUDI Wallet rejects a COSE-based Referenced Token where the "idx" field within "status_list" has an invalid type (negative or wrong CBOR type).

## References
[Token Status List (TSL) draft-20] Section 6.3

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Referenced Token includes the Status element that contains the Status_list element (index and URI)
C) The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
D) The Referenced Token contains the "idx" field within "status_list" set to an invalid type (negative or wrong CBOR type).

## Test Scenario
1. Verify the Wallet's handling of the invalid "idx" type.

## Expected results
1. The Referenced Token is rejected.

