# WS_RP_MS_Metadata_097

## Objective
Verify that the EUDI Wallet rejects a COSE-based Referenced Token where the required "idx" field within "status_list" is missing.

## References
[Token Status List (TSL) draft-20] Section 6.3

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
C) The Referenced Token does not contain the "idx" field within "status_list".

## Test Scenario
1. Verify the Wallet's handling of the missing "idx" field.

## Expected results
1. The Referenced Token is rejected.

