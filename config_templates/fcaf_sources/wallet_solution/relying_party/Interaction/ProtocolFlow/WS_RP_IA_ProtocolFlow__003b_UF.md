# WS_RP_IA_ProtocolFlow_003b

## Objective

Verify that the Wallet, in the presentation flow via DC API using OpenID4VP, does not provide a response with the Response mode `dc_api`.

## References

- [HAIP] section 5.2
- [OpenID4VP] section 8.5, A.2
- [DC API]

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation via the W3C Digital Credentials API or equivalent platform API.
Presentation using OpenID4VP

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. End-user interacts with Verifier using a User-agent.

## Test Scenario

1. The Verifier provides a OpenID4VP Request Object to the User-agent for usage with the W3C DC API or equivalent platform API, where the Request Object:
    1. is requesting presentation of `default_credential_A`,
    2. has the `response_mode` parameter set to the value `dc_api`,
    3. having all other required parameters with valid values,
    4. and valid signed.
2. End-user triggers the mediating API to initiate the provided Request Object.
3. Verify the Authorization Response received.

## Expected results

1. User-agent presents option to End-user to engage presentation.
2. The Wallet is invoked.
3. Wallet does not complete the presentation interaction, and
    1. informs, if applicable, the user of an invalid request, and
    2. aborting the interaction with the Verifier, either by
        1. responding with an error `invalid_request`, or
        2. responding by returning an error without any details, or
        3. discontinuing the interaction, e.g. by closing the communication channel, if applicable.
    3. if any response is shared to the Verifier, the response does not include a `vp_token`.
