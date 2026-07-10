# WS_RP_SM_TrustMechanisms_021

## Objective
Verify that Wallet can handle Authorization Request using authorization key identifier mechanism to support multiple X.509-based trust mechanism supported by the Wallet profile.

## References
[OIDF.HAIP] section 5 (introduction)
[OIDF.OID4VP] section 6.1.1.1, section 8.5

## Profile applicability
multiple X.509-based trust mechanisms supported by the Wallet

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends Credential Query to the Wallet using Authority Key Identifier (aki)-based Trusted Authority Query (trusted_authorities) for DCQL.
3. `aki` array includes a collection of encoded Key Identifiers from the certificates 
4. Verify that Wallet does not answer with error `invalid_request`, which can be sent when the Wallet does not support the Client Identifier Prefix passed in the Authorization Request.
5. Verify if Wallet asks for user consent to present the Credential.
6. User gives consent.
7. Verify if Wallet answers Credential Query successfully.

## Expected results
1. This is the case.
2. This is the case.
3. This is the case.
4. Wallet does not answer with error `invalid_request`, which can be sent when the Wallet does not support the Client Identifier Prefix passed in the Authorization Request.
5. Wallet asks for user consent.
6. This is the case.
7. Wallet answers Credential Query successfully

