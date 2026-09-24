# WS_RP_MS_CredentialFormats_045a

## Objective

Verify that the Wallet syntactically correctly serializes a presentation of a credential in SD-JWT VC format, when the issuer is using compact serialization, the credential has key binding and using OpenID4VP.

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

## Test Scenario

1. Verify the presentation value in the `vp_token`.

## Expected results

1. The presentation value is a string consisting of:
    1. a signed JWT, that is a series of 3 sets of base64url-encoded data concatenated by period ('.'),
    2. optionally, when including disclosure(s):
        1. followed by a tilde ('~'),
        2. followed by disclosures, that are series of base64url-encoded data, concatenated by a tilde ('~'),
    3. followed by a tilde ('~'),
    4. followed by another signed JWT, that is a series of 3 sets of base64url-encoded data concatenated by period ('.').
