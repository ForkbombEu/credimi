# WS_RP_MS_ProtocolMessages_010

## Objective
Verify that the Wallet does not process the request if client_id is missing from the Request Object.

## References
[RFC9101]; [OIDF.OID4VP] section 5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet holds at least one credential that can match a valid DCQL query, and a second scenario where no matching credential is available.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends a Request Object where client_id is absent.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet rejects the request and returns an invalid_request error indicating missing client_id; presentation flow is not initiated.

