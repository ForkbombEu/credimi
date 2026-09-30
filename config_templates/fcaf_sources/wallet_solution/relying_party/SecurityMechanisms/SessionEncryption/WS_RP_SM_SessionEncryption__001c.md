# WS_RP_SM_SessionEncryption_001c

## Objective

Verify that the Wallet encrypts the Authorization Response using an appropriate algorithm to encrypt or determine the value of the CEK, when presenting (a) credential(s) using OpenID4VP.

## References

- [ETSI TS 119 472-2] section 6.3.3
- [HAIP] section 5
- [OpenID4VP] section 8.3
- [RFC7516] section 4.1.1
- [RFC7518] section 4.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been triggered.
3. The Verifier shared an emphemeral key as part of the Authorization Request
4. The Wallet successfully transmitted an Authorization Response to the Verifier in response to the Authorization Request.
5. The Wallet provided a correctly serialized encrypted Authorization Response.

## Test Scenario

1. Verify the algorithm used to encrypt or determine the value of the CEK of the encrypted Authorization Response is appropriate.

## Expected results

1. The JWE `alg` header parameter has the value `ECDH-ES` and equals the value of the `alg` property of the selected Verifier JWK.
