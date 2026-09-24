# WS_RP_SM_DeviceBinding_012b

## Objective

Verify that the Wallet includes a valid signed key binding JWT cryptographically bound to the credential, if a presented credential in SD-JWT VC format is key bound.

## References

- [HAIP] section 6.1
- [OpenID4VP] section 8, B.3.6
- [SD-JWT VC] section 3.4
- [RFC9901] section 4.1.2, 7.3
- [RFC7515]

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
6. The SD-JWT has a top-level `cnf` claim.
7. The presentation value of the credential contains a correctly serialized KB-JWT.
8. The KB-JWT uses an acceptable signature algorithm.

## Test Scenario

1. Verify the KB-JWT signature using the public key identified by the `cnf` claim in the presented SD-JWT.

## Expected results

1. The signature is valid.
