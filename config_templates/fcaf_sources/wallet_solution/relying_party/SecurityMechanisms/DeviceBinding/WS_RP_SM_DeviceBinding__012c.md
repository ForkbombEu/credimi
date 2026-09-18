# WS_RP_SM_DeviceBinding_012c

## Objective

Verify that the Wallet contains a key binding JWT bound to the presented SD-JWT VC, if a presented credential in SD-JWT VC format is key bound.

## References

- [HAIP] section 6.1
- [OpenID4VP] section 8, B.3.6
- [SD-JWT VC] section 3.4
- [RFC9901] section 4.3.1, 7.3

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Credential in SD-JWT VC format
Credential with key-binding
Issuer uses compact serialization of SD-JWT and SD-JWT VC

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Verifier requested a credential to be presented, using a valid, trusted request for 'default_credential_A', which is key bound, and requesting presentation in SD-JWT VC format.
3. The Wallet transmitted a syntactically correct presentation in the `vp_token` of the Authorization Response.
4. The presentation value of the credential is a syntactically correct serialization in compact serialization format of an SD-JWT VC.
5. The presentation value of the credential contains a valid signed SD-JWT.
6. The presentation value of the credential contains a valid signed KB-JWT.

## Test Scenario

1. Verify the value of the `sd_hash` claim of the KB-JWT.

## Expected results

1. The value of `sd_hash` claim of the KB-JWT is equal to the base64url-encoded digest over the presented SD-JWT.
