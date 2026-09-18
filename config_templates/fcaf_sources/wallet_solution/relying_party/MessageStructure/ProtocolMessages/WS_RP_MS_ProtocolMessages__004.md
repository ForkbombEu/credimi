# WS_RP_MS_ProtocolMessages_004

## Objective

Verify that the Wallet, when interacting using OpenID4VP, accepts an Authorization Request sent as a Request Object where the `typ` header parameter has the correct value of `oauth-authz-req+jwt`.

## References

- [OpenID4VP] section 5
- [RFC9101]

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Wallet and Verifier are engaged, and a presentation using the OpenID4VP protocol has been triggered.

## Test Scenario

1. Verifier sends an Authorization Request containing a Signed Request Object, where the Request Object:
    1. contains all minimal required parameters to request 'default_credential_A',
    2. is valid, including a valid signature,
    3. has a `typ` header parameter with the value `oauth-authz-req+jwt`.

## Expected results

1. Wallet responds with a presentation of 'default_credential_A'.
