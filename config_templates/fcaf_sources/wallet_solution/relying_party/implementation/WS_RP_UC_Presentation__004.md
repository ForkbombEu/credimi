# WS_RP_UC_Presentation_004

## Objective
Verify that when an Authorization Request is rendered on Device A using the invocation mechanism mandated by the applicable profile (e.g. a custom URL scheme, universal link, or QR code encoding such a URI), the Wallet on Device B is able to receive and process the request.

## References
[OIDF.OID4VP] section 5.7

## Profile applicability
Wallet supports DC API cross-device flow (Appendix A); HAIP/ETSI profile defines cross-device requirements

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction on Device A (Verifier device), which renders the Authorization Request using the invocation mechanism mandated by the applicable profile.
2. User transfers the Authorization Request from Device A to Device B (Wallet device) using the rendered invocation mechanism.
3. Wallet on Device B receives the Authorization Request via its registered handler for the mandated scheme.
4. Wallet parses and validates the Authorization Request.
5. Wallet performs the presentation flow on Device B.

## Expected results
1. Authorization Request is successfully rendered on Device A using the profile-mandated mechanism.
2. The request is successfully transferred to Device B.
3. Wallet on Device B successfully receives the Authorization Request via its registered handler.
4. Wallet successfully parses and validates the Authorization Request.
5. Wallet completes the presentation flow on Device B and returns the Authorization Response to the Verifier.

