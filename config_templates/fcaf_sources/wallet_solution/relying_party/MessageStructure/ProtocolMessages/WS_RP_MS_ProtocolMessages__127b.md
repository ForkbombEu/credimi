# WS_RP_MS_ProtocolMessages_127b

## Objective

Verify that the Wallet's Authorization Response has syntactically correct presentation(s) in each entry in a `vp_token`, if presenting (a) credential(s) in response to a request for (a) credential(s), using OpenID4VP.

## References

- [HAIP] section 5
- [OpenID4VP] section 8.1, B.2.5, B.3.6

## Profile applicability

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation using OpenID4VP

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Verifier requested credential(s) to be presented, using a valid, trusted request for credential(s).
3. The Authorization Response from the Wallet is received by the Verifier.
4. The Authorization Response is successfully decrypted by the Verifier.
5. The Authorization Response contains a syntactically correct `vp_token` parameter.

## Test Scenario

1. Verify the value(s) of the presentation(s) in the `vp_token`.

## Expected results

1. Each value of an entry in the `vp_token` parameter of the Authorization Response:
    1. is an array,
    2. each element in that array is a string or an JSON object.
