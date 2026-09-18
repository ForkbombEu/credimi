# WS_RP_IA_MainInteraction_034b

## Objective

Verify that the Wallet responds to a DCQL-query for two data elements with both data elements present, and does not include other data elements, for a credential in mdoc format.

## References

- [OpenID4VP] section 7.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Credential in mdoc format

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been initiated.

## Test Scenario

1. Verifier sends a request with a valid DCQL query, with a `credentials` property that is an array containing one object with the following properties:
    1. the `format` property with the value `mso_mdoc`.
    2. the `claims` property, as an array with:
        1. one object containing a `path` property whose value is an array containing two elements that are a string, with as values the namespace and identifier of a data element in 'default_credential_A'.
        2. one object containing a `path` property whose value is an array containing two elements that are a string, with as values the namespace and identifier of another data element in 'default_credential_A'.
    3. the `meta` property, containing a `doctype_value` property that is a string with as value the doctype of 'default_credential_A'.
2. Verify the received presentation.

## Expected results

1. The Wallet responds with a presentation of 'default_credential_A'.
2. The presentation has the following characteristics:
    1. The presentation is for a credential in mdoc format, and has the doctype as requested.
    2. The presentation contains both requested data elements.
    3. The presentation does not contain any other data elements in the mdoc credential.
