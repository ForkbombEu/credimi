# WS_RP_MS_ProtocolMessages_013

## Objective
Verify that the Wallet successfully evaluates a valid dcql_query that matches multiple credentials and presents the user with all candidate credentials for selection.

## References
[OIDF.OID4VP] sections 5, 6.4; [ISO 18013-7] annex C

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet contains a credential matching the request.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a valid dcql_query that matches multiple credentials held by the Wallet.
3. Wallet evaluates the DCQL query and identifies multiple candidate credentials.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives and parses the Authorization Request successfully.
3. Wallet presents the user with all matching credentials for selection; presentation flow proceeds after user selection.

