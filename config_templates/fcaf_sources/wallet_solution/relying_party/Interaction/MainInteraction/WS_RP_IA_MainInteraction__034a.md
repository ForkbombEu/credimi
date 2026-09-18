# WS_RP_IA_MainInteraction_034a

## Objective

Verify that the Wallet responds to a DCQL-query for two selectively disclosable claims with disclosure of both claims, and does not include other selectively disclosable claims, for a credential in SD-JWT VC format.

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
        1. one object containing a `path` property with as value an array containing one element that is a string, with the name of a top-level claim that is selectively disclosable in 'default_credential_A'.
        2. one object containing a `path` property with as value an array containing one element that is a string, with the name of another top-level claim that is selectively disclosable in 'default_credential_A'.
    3. the `meta` property, containing a `vct_values` property with as value an array containing one element that is a string with the verifiable credential type of 'default_credential_A'.
2. Verify the received presentation.

## Expected results

1. The Wallet responds with a presentation of 'default_credential_A'.
2. The presentation has the following characteristics:
    1. The presentation is for a credential in SD-JWT VC format, of the verifiable credential type and with the `vct` claim as requested.
    2. The presentation contains the disclosures of both claims requested, as top-level claims in the verifiable credential.
    3. The presentation does not contain any other disclosures for claims that allow for selective disclosure in the SD-JWT VC credential.
