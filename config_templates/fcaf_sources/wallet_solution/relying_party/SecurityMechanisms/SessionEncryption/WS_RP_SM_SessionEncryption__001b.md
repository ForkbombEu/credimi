# WS_RP_SM_SessionEncryption_001b

## Objective

Verify that the Wallet encrypts the Authorization Response using an appropriate encryption algorithm, when presenting (a) credential(s) using OpenID4VP.

## References

- [ETSI TS 119 472-2] section 6.3.3
- [HAIP] section 5
- [OpenID4VP] section 8.3
- [RFC7516] section 4.1.2
- [RFC7518] section 5.1, 5.3

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

1. Verify the encryption algorithm used for encrypting the encrypted Authorization Response.

## Expected results

1. The algorithm for the content encryption (JWE's `enc`) is either `A256GCM` or `A128GCM`.
