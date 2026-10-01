# WS_RP_SM_DeviceBinding_012a

## Objective

Verify that the Wallet includes a key binding JWT with an acceptable signature algorithm, if a presented credential in SD-JWT VC format is key bound.

## References

- [HAIP] section 6.1
- [OpenID4VP] section 8, B.3.6
- [SD-JWT VC] section 3.4
- [RFC9901] section 4.1.2, 7.3
- [RFC7515] section 4.1.1
- [ECCG ACM] section 5.2

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
5. The presentation value of the credential contains a correctly serialized KB-JWT.

## Test Scenario

1. Verify the signature algorithm (`alg` in protected header) used for the KB-JWT.

## Expected results

1. The signature algorithm in `alg` is on the list of acceptable algorithms [ECCG ACM], and does not have the value `none`.

## Comments

- Note that not all acceptable algorithms in [ECCG ACM] are standardized for usage with JOSE. See IANA registry on JOSE algorithms for standardized algorithm identifiers and their specifications.
