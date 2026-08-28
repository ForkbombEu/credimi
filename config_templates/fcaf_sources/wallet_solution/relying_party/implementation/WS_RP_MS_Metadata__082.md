# WS_RP_MS_Metadata_082

## Objective
Verify that the EUDI Wallet handles a JOSE-based Referenced Token without a "status" claim according to local policy.

## References
[Token Status List (TSL) draft-20] Section 6.2

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in JWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B)The Issuer has provided a JOSE-based Referenced Token to the EUDI Wallet.
C) The Referenced Token does not contain the "status" claim.

## Test Scenario
1. Verify the Wallet's handling of the absent "status" claim.

## Expected results
1. The Wallet handles the absent "status" claim according to local policy.

