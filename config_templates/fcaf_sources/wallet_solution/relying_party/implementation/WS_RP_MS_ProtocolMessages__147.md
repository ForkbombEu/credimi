# WS_RP_MS_ProtocolMessages_147

## Objective
Test the Wallet returns the access_denied error message when the Wallet does not have the requested Credentials to satisfy the Authorization Request.

## References
[OID4VP Section 8.5]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with verifier.
2. The Verifier sends an Authorization request for credentials the wallet does not have.
3. Wallet processes request.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The Wallet does NOT proceed to credential selection or user authorization.
4. Verify the Wallet returns an error response, where the error parameter is exactly access_denied.

