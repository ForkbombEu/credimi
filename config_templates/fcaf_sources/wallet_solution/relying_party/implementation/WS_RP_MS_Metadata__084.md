# WS_RP_MS_Metadata_084

## Objective
Verify that the EUDI Wallet handles a JOSE-based Referenced Token where the "status" claim does not contain a "status_list" object.

## References
[Token Status List (TSL) draft-20] Section 6.2

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in JWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Issuer has provided a JOSE-based Referenced Token to the EUDI Wallet.
C) The Referenced Token contains the "status" claim but without a "status_list" object.

## Test Scenario
1. Verify the Wallet's handling of the absent "status_list" within the "status" claim.

## Expected results
1.  The Wallet rejects the Referenced Token when there is no status_list JSON object present in the status JWT Claims Set

