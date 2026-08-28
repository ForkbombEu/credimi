# WS_RP_MS_CredentialFormats_041

## Objective
Verify that when Wallet is presenting several ISO mdocs, each ISO mdoc is returned in a separate DeviceResponse (as defined in 8.3.2.1.2.2 of [ISO.18013-5]), each matching to a respective DCQL query. Therefore, the resulting vp_token contains multiple DeviceResponse instances.

## References
[OIDF.HAIP] section 5.3.1
[OIDF.OID4VP] appendix B.2 (ISO mdoc profile)
[ISO.18013-5] section 8.3.2.1.2.2. 

## Profile applicability
Wallet supports ISO mdoc

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends 3 DCQL query to the Wallet. 
3. Verifier requests presentation in mdoc format.
4. Verify if Wallet can handle receiving multiple DCQL queries for Credentials in ISO mdoc format.
5. Verify if Wallet asks for user consent to present the Credential.
6. User gives consent.
7. Verify if `vp_token` contains multiple `DeviceResponse` instances.
8. Verify if for each query, Wallet returns one Device Response matching each DCQL query.

## Expected results
1. This is the case.
2. This is the case.
3. This is the case.
4. Wallet receives multiple DCQL queries for Credentials in ISO mdoc format successfully.
5. Wallet asks for user consent.
6. This is the case.
7. `vp_token` contains multiple `DeviceResponse` instances.
8. For each query, Wallet returns one Device Response matching each DCQL query.

