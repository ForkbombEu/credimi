# WS_RP_IA_MainInteraction_034f

## Objective

Verify that the Wallet responds to a DCQL-query selecting the first element of a top-level array claim, and does not include other selectively disclosable claims, for a credential in SD-JWT VC format.

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
    2. the `claims` property, as an array with one object containing a `path` property with as value an array containing two elements,
        1. the first a string, with the name of a top-level claim that is an array of which the elements are selectively disclosable in 'default_credential_A'.
        2. the second the integer `0`.
    3. the `meta` property, containing a `vct_values` property with as value an array containing one element that is a string with the verifiable credential type of 'default_credential_A'.
2. Verify the received presentation.

## Expected results

1. The Wallet responds with a presentation of 'default_credential_A'.
2. The presentation has the following characteristics:
    1. The presentation is for a credential in SD-JWT VC format, of the verifiable credential type and with the `vct` claim as requested.
    2. The presentation contains the disclosure of the first element of the selected top-level array claim.
    3. The presentation does not contain any other disclosures for claims that allow for selective disclosure in the SD-JWT VC credential.
