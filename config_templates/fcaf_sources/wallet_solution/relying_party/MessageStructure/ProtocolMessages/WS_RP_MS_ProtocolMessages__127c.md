# WS_RP_MS_ProtocolMessages_127c

## Objective

Verify that the Wallet's Authorization Response has (a) syntactically correct SD-JWT VC presentation(s) in a `vp_token`, if presenting (a) credential(s) in response to a request for (a) credential(s), using OpenID4VP and presenting a SD-JWT VC credential.

## References

- [HAIP] section 5, 6.1
- [OpenID4VP] section 8.1, B.3.6
- [SD-JWT VC] section 4.1
- [RFC9901] section 4, 8

## Profile applicability

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation using OpenID4VP
Credential in SD-JWT VC format

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Verifier requested credential(s) to be presented, using a valid, trusted request for credential(s) in SD-JWT VC format.
3. The Authorization Response from the Wallet is received by the Verifier.
4. The Authorization Response is successfully decrypted by the Verifier.
5. The Authorization Response contains a syntactically correct `vp_token` parameter.
6. The `vp_token` contains syntactically correct presentation(s).

## Test Scenario

1. Verify the value(s) of the presentation(s) in the `vp_token`.

## Expected results

1. Each value in each presentation in the `vp_token` parameter of the Authorization Response is either
    1. a string, that has as value a syntactically correct compact serialization ([SD-JWT] section 4) of either a SD-JWT or a SD-JWT+KB,
    2. a JSON object, that is a syntactically correct JWS JSON serialization ([SD-JWT] section 8) of either a SD-JWT or a SD-JWT+KB.

## Comments

- Note: the syntactically correct here can be limited to basic notation checks, since the contents and its structure will be covered in other tests. Syntactically therefor here is
    - a series of base64url-encoded data concatenated by periods ('.') and tilde ('~'), for compact serialization.
    - a JSON object, for JWS JSON serialization.
