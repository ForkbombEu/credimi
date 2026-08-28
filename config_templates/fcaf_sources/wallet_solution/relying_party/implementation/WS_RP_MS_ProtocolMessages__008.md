# WS_RP_MS_ProtocolMessages_008

## Objective
Verify that the Wallet successfully processes the Authorization Request when client_id is present but iss claim is absent.

## References
[RFC9101]; [OIDF.OID4VP] section 5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet holds at least one credential that can match a valid DCQL query, and a second scenario where no matching credential is available.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object containing client_id but NO iss claim.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet processes the Authorization Request successfully; Verifier is identified by the client_id claim.

