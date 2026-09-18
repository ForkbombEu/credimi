# WS_RP_SM_SessionEncryption_001a

## Objective

Verify that the Wallet correctly serializes an encrypted Authorization Response (JWE compact serialization), when presenting (a) credential(s) using OpenID4VP.

## References

- [ETSI TS 119 472-2] section 6.3.3
- [HAIP] section 5
- [OpenID4VP] section 8.3
- [RFC7519] section 3
- [RFC7516] section 7.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been triggered.
3. The Verifier shared an ephemeral key as part of the Authorization Request.
4. The Wallet successfully transmitted an Authorization Response to the Verifier in response to the Authorization Request.

## Test Scenario

1. For the Authorization Response, perform all Shared_JWT_JWE test cases.

## Expected results

1. All test cases pass.
