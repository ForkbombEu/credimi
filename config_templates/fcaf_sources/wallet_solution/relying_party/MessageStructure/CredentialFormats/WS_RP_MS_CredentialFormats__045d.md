# WS_RP_MS_CredentialFormats_045d

## Objective

Verify that the Wallet correctly serializes the key binding JWT in a presentation of a credential in SD-JWT VC format, when the issuer is using compact serialization, the credential has key binding and using OpenID4VP.

## References

- [HAIP] section 6.1, 6.1.1.1
- [SD-JWT VC] section 4.1
- [RFC9901] section 4

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation using OpenID4VP
Credential in SD-JWT VC format
Credential with key-binding
Issuer uses compact serialization of SD-JWT and SD-JWT VC

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Verifier requested a credential to be presented, using a valid, trusted request for 'default_credential_A' (that is key-bound) and requesting presentation in SD-JWT VC format of all claims.
3. The Wallet transmitted a syntactically correct presentation in the `vp_token` of the Authorization Response.
4. The presentation value of the credential is a syntactically correct serialization in compact serialization format of an SD-JWT VC.

## Test Scenario

1. Extract the KB-JWT part of the presentation value, that is the part after to not including the last tilde ('~') of the presentation, consisting of a series of 3 base64url-encoded data concatenated by period ('.').
2. Perform all Shared_JWT_JWS test cases on the extract KB-JWT part.

## Expected results

1. The key binding part contains a compact serialized Signed JWT (i.e. base64.base64.base64 pattern).
2. All test cases pass.
