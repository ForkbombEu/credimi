# WS_RP_MS_ProtocolMessages_009

## Objective
Verify that the Wallet successfully processes the Authorization Request when both client_id and iss are present, but their values differ.

## References
[RFC9101]; [OIDF.OID4VP] section 5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet holds at least one credential that can match a valid DCQL query, and a second scenario where no matching credential is available.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends a Request Object containing both client_id AND iss claim, with iss value different from client_id.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet processes the Authorization Request successfully; iss claim is silently ignored; Verifier is identified by the client_id claim only.

