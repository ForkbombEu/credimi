# WS_RP_MS_ProtocolMessages_014

## Objective
Verify that the Wallet returns an error and does not initiate the presentation flow when no credential satisfies the dcql_query.

## References
[OIDF.OID4VP] sections 5, 6.4; [ISO 18013-7] annex C

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet does not hold a matching credential.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a valid dcql_query that cannot be satisfied by any credential held by the Wallet.
3. Wallet evaluates the DCQL query and finds no matching credential.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives and parses the Authorization Request successfully.
3. Wallet informs the user that no suitable credential is available and returns an access_denied error to the Verifier; presentation flow is not initiated.

