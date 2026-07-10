# WS_RP_IA_Metadata_012

## Objective
Verify that when the Wallet receives a Request Object using Static Discovery metadata with the aud claim set to the SIOPv2 static identifier "https://self-issued.me/v2", it accepts and processes the request.

## References
[OIDF.OID4VP] section 5.8

## Profile applicability
static discovery

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using Static Discovery metadata with the aud claim set to "https://self-issued.me/v2".
3. Wallet parses the Request Object.
4. Wallet validates the aud claim against the expected SIOPv2 static identifier.
5. Wallet proceeds with request processing.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet validates that the aud claim equals "https://self-issued.me/v2".
5. Wallet accepts and processes the request; the flow proceeds normally.

