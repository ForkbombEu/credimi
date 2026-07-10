# WS_RP_MS_ProtocolMessages_151

## Objective
Test the Wallet returns the error message vp_formats_not_supported when the Wallet does not support any of the formats requested.
(mso_mdoc case)

## References
[OID4VP Section 8.5]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet supports vc+sd-jwt, but does NOT support mso_mdoc

## Test Scenario
1. The Wallet engages with verifier.
2. The verifier sends an Authorization request where the vp_formats parameter only includes mso_mdoc.
3. Wallet processes request.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. The Wallet identifies it cannot fulfill request due to format requirement and does NOT show credential selection screen.
4. Verify the Wallet returns an error response, where the error parameter is exactly vp_formats_not_supported.

