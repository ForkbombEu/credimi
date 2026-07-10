# WS_RP_MS_ProtocolMessages_029

## Objective
Verify that when a verifier_info attestation omits the credential_ids field, the Wallet applies the attestation to all Credentials requested in the DCQL query.

## References
[OIDF.OID4VP] Section 5.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a verifier_info attestation with no credential_ids field.
3. Wallet parses the attestation.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives and parses the Authorization Request.
3. Wallet applies the attestation to all Credentials requested in the dcql_query; presentation flow proceeds.

