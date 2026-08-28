# WS_RP_MS_Metadata_094

## Objective
Verify that the EUDI Wallet handles a COSE-based Referenced Token where the Status Map does not contain a "status_list" entry.

## References
[Token Status List (TSL) draft-20] Section 6.3

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
C) The Referenced Token contains the Status Map present but without a "status_list" entry.

## Test Scenario
1. Verify the Wallet's handling of the absent "status_list" within the Status Map.

## Expected results
1. The Wallet handles the missing "status_list" according to local policy.

