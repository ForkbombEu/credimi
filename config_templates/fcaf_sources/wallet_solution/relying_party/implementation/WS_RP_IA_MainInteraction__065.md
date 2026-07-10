# WS_RP_IA_MainInteraction_065

## Objective
Verify that Wallet supports receiving a Credential Query from the Verifier in DCQL language, including the object `trusted_authorities` (aki)-based as defined in section 6.1.1.1 of [OIDF.OID4VP].

## References
[OIDF.HAIP] section 5 (introduction)
[OIDF.OID4VP] section 6.1.1.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends an Authorization Request containing a DCQL credential query that specifies an Authority Key Identifier (AKI) inside the trusted_authorities object.
3. The wallet processes the DCQL query and validates the requested trusted authorities against the credentials issued to the wallet.
4. The wallet prompts the user for consent to present the matching credential.
5. The user provides consent.
6. The wallet transmits the Authorization Response back to the verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives the request.
3. Verify that the Wallet correctly parses the trusted_authorities query without throwing a processing error or returning an invalid_request.
4. Verify that the Wallet identifies the matching credential and presents consent to the user.
5. User consent is captured successfully.
6. Verify that the Wallet compiles the requested payload and successfully transmits the response to the verifier.