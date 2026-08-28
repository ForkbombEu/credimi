# WS_RP_SM_RpIntegrity_CryptographicSignature002

## Objective
Verify that Wallet does not support RSASSA-PSS using SHA-384 and MGF1 with SHA-384 (RS384) for validating signed presentation requests.

## References
[OIDF.HAIP] section 8

## Profile applicability
JOSE algorithm identifier ES257

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential.
3. Verify that Wallet receives signed request using RSASSA-PSS using SHA-384 and MGF1 with SHA-384 (RS384).from the Verifier.
4. Wallet processes the request.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet receives signed request using RSASSA-PSS using SHA-384 and MGF1 with SHA-384 (RS384).
4. Wallet rejects the transaction and either:
a. returns specific error `invalid_client`,
b. returns general error,
c. only discontinues the transaction.

