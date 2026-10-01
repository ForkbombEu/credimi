# WS_RP_SM_SessionEncryption_001e

## Objective

Verify that the encrypted Authorization Response from the Wallet can be decrypted, when presenting (a) credential(s) using OpenID4VP.

## References

- [ETSI TS 119 472-2] section 6.3.3
- [HAIP] section 5
- [OpenID4VP] section 8.3
- [RFC7516] section 5.2
- [RFC7518] section 4.6, 5.3

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been triggered.
3. The Verifier provided an emphemeral key as part of (client metadata of) the Authorization Request.
4. The Wallet successfully transmitted an Authorization Response to the Verifier in response to the Authorization Request.
5. The Wallet provided a correctly serialized encrypted Authorization Response.
6. The Wallet uses an appropriate algorithm to encrypt or determine encrypt or determine the value of the CEK of the encrypted Authorization Response.
7. TODO check curve.
8. The Wallet uses an appropriate encryption algorithm for encryption of the encrypted Authorization Response.

## Test Scenario

1. Verify the encrypted Authorization Response can be decrypted.

## Expected results

1. The Authorization Response can be decrypted, using the emphemeral key of the Authorization Request.
