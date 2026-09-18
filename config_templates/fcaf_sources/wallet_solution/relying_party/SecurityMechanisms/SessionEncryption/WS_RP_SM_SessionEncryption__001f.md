# WS_RP_SM_SessionEncryption_001f

## Objective

Verify that the Wallet correctly encrypted the Authorization Response as an unsigned JWT, when presenting (a) credential(s) using OpenID4VP.

## References

- [ETSI TS 119 472-2] section 6.3.3
- [HAIP] section 5
- [OpenID4VP] section 8.3
- [RFC7519]

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been triggered.
3. The Verifier shared an ephemeral key as part of the Authorization Request.
4. The Wallet successfully transmitted an Authorization Response to the Verifier in response to the Authorization Request.
5. The Authorization Response is a correctly serialized encrypted Authorization Response.
6. The Authorization Response can be decrypted.

## Test Scenario

1. Perform all Shared_JSON test cases on the decrypted Authorization Response.

## Expected results

1. All test cases pass.

## Comments

- The contents of the Authorization Response JWT are verified in MessageStructure ProtocolMessages tests.
