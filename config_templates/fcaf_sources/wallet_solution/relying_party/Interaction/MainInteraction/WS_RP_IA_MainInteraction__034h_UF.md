# WS_RP_IA_MainInteraction_034h_UF

## Objective

Verify that the Wallet does not present a credential in response to a DCQL-query requesting two claims, of which one is selectively disclosable in the held credential and the other is not present in it, for a credential in SD-JWT VC format.

## References

- [OpenID4VP] section 7.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Credential in SD-JWT VC format

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been initiated.

## Test Scenario

1. Verifier sends a request with a valid DCQL query, with a `credentials` property that is an array containing one object with the following properties:
    1. the `format` property with the value `dc+sd-jwt`.
    2. the `claims` property, as an array with:
        1. one object containing a `path` property whose value is an array containing one element that is a string, with the name of a top-level claim that is selectively disclosable in 'default_credential_A'.
        2. one object containing a `path` property whose value is an array containing one element that is a string, with a value that is not the name of any claim in 'default_credential_A'.
    3. the `meta` property, containing a `vct_values` property whose value is an array containing one element that is a string with as value the verifiable credential type of 'default_credential_A'.
2. Verify the received presentation.

## Expected results

1. The Wallet does not complete the presentation interaction, and
    1. informs, if applicable, the user of not having the requested credential as it does not have the requested attributes, and
    2. aborting the interaction with the Verifier, either by
        1. responding with an error `access_denied`, or
        2. responding by returning an error without any details, or
        3. discontinuing the interaction, e.g. by closing the communication channel, if applicable.
    3. if any response is shared to the Verifier, the response does not include a `vp_token`.
