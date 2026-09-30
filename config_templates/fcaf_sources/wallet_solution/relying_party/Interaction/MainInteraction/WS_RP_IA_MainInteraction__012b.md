# WS_RP_IA_MainInteraction_012b

## Objective

Verify that the Wallet responds to a DCQL-query for a single claim with disclosure of that data element, and does not include other data elements, for a credential in mdoc format.

## References

- [HAIP] section 5, 5.3.1
- [OpenID4VP] section 6.1, 6.3, B.2.3

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
    2. the `claims` property, as an array with one object containing a `path` property with as value an array containing two elements that are a string, with as values the namespace and identifier of a data element in 'default_credential_A'.
    3. the `meta` property, containing a `doctype_value` property with as value a string with the doctype of 'default_credential_A'.
2. Verify the received presentation.

## Expected results

1. The Wallet responds with a presentation of 'default_credential_A'.
2. The presentation has the following characteristics:
    1. The presentation is for a document in mdoc format, of the document type and with the `docType` as requested.
    2. The presentation contains the data element requested.
    3. The presentation does not contain any other data elements of the mdoc document.
