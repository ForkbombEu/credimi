# WS_RP_IA_MainInteraction_012d_UF

## Objective

Verify that the Wallet does not present a credential in response to a DCQL-query for a single data element not present in any matching credential in mdoc format.

## References

- [HAIP] section 5, 5.3.1
- [OpenID4VP] section 6.1, 6.3, 6.4, 8.5, B.2.3

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been initiated.

## Test Scenario

1. Verifier sends a request with a valid DCQL query, with a `credentials` property that is an array containing one object with the following properties:
    1. the `format` property with the value `mso_mdoc`.
    2. the `claims` property, as an array with one object containing a `path` property with as value an array containing two elements that are a string, with as values the namespace and identifier of a data element that is not present in 'default_credential_A'.
    3. the `meta` property, containing a `doctype_value` property with as value a string with the doctype of 'default_credential_A'.

## Expected results

1. The Wallet does not complete the presentation interaction, and
    1. informs, if applicable, the user of not having the requested credential, and
    2. aborting the interaction with the Verifier, either by
        1. responding with an error `access_denied`, or
        2. responding by returning an error without any details, or
        3. discontinuing the interaction, e.g. by closing the communication channel, if applicable.
    3. if any response is shared to the Verifier, the response does not include a `vp_token`.
