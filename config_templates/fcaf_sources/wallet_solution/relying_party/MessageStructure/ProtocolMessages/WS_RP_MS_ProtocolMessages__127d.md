# WS_RP_MS_ProtocolMessages_127d

## Objective

Verify that the Wallet's Authorization Response has (a) syntactically correct mdoc presentation(s) in a `vp_token`, if presenting (a) credential(s) in response to a request for (a) credential(s), using OpenID4VP and presenting a mdoc credential.

## References

- [HAIP] section 5
- [OpenID4VP] section 8.1, B.2.5

## Profile applicability

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation using OpenID4VP
Credential in mdoc format

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Verifier requested credential(s) to be presented, using a valid, trusted request for credential(s) in mdoc format.
3. The Authorization Response from the Wallet is received by the Verifier.
4. The Authorization Response is successfully decrypted by the Verifier.
5. The Authorization Response contains a syntactically correct `vp_token` parameter.
6. The `vp_token` contains syntactically correct presentation(s).

## Test Scenario

1. Verify the value(s) of the presentation(s) in the `vp_token`.

## Expected results

1. Each value in each presentation in the `vp_token` parameter of the Authorization Response is a string which value contains correct base64url-encoded data.
