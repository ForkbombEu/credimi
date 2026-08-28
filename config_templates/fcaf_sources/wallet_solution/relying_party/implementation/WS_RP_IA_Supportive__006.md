# WS_RP_IA_Supportive_006

## Objective
Test the wallet returns the error message: wallet_unavailable when the device the wallet is located on is under extreme load and hence is out of RAM/ disk space.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Device is under extreme load

## Test Scenario
1. The wallet engages with verifier.
2. The verifier sends a valid Authorization request.
3. Wallet processes request
4. Test the response returned by the wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request.
3. Wallet cannot process request due to RAM/disk space.
4. Verify the Wallet does NOT return a credential; instead, it returns an error response where the error parameter is exactly wallet_unavailable.

