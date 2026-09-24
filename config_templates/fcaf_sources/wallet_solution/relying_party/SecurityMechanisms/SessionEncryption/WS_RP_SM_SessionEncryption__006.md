# WS_RP_SM_SessionEncryption_006

## Objective

Verify that Wallet raises an error if there is no `jwk` within `client_metadata` sent by the Verifier with `alg` value set to ECDH-ES.

## References

- [HAIP] Section 5 (introduction)
- TODO: [OpenID4VP] Section 8.3 The alg parameter MUST be present in the JWKs
- [RFC7516] Section 4.1.1
- [RFC7518] Section 6.2.1.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

None

## Test Scenario

1. Verify that Wallet raises an error if there is no "jwk" within "client_metadata" sent by the Verifier with "alg" value set to ECDH-ES.

## Expected results

1. Wallet raises an error.
