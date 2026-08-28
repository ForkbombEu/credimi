# WS_RP_IA_MainInteraction_042

## Objective
Test the wallet does NOT include entries in the vp_token, for optional Credential Queries, when there are no matching Credentials.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet does NOT contain credential A.
Wallet does contain credential B.

## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends a request where credential query includes an optional credential A, along with a non-optional credential B.
3. The wallet returns a vp_token parameter in its response

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request, and user gives permission.
3. Verify the vp_token entries contains only credential B