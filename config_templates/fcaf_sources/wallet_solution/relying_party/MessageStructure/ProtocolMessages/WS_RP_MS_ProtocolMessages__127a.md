# WS_RP_MS_ProtocolMessages_127a

## Objective

Verify that the Wallet's Authorization Response has a syntactically correct `vp_token`, if presenting (a) credential(s) in response to a request for (a) credential(s), using OpenID4VP.

## References

- [ETSI TS 119 472-2] section 6.3.3
- [HAIP] section 5
- [OpenID4VP] section 8.1, 8.3

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

## Test Scenario

1. Verify the presentation response.

## Expected results

1. The decrypted Authorization Response contains a `vp_token` parameter. The `vp_token` parameter is a JSON object, with:
    1. each entry in the object has a key that is a string,
    2. each entry in the object has a value that is an array.
