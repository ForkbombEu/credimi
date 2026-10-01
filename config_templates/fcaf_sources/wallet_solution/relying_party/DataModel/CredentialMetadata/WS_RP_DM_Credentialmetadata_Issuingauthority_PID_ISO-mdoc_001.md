# WS_RP_DM_Credentialmetadata_Issuingauthority_PID_ISO-mdoc_001

## Objective

This test case verifies that the data element `issuing_authority` is present in the mdoc data. Note that `issuing_authority` is the Attribute Identifier in ISO-mdoc for the Data Identifier `issuing_authority`.

## References

- [PID rulebook] Annex 3.01, Section 4.1 (Table 2)

## Profile applicability

The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `issuing_authority` is present in the mdoc data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.

## Test Scenario

1. Verify the presence of a data element with identifier `issuing_authority` in the device retrieval mdoc response.
2. Verify the absence of an `ErrorItem` for data element `issuing_authority`.

## Expected results

1. One data element with identifier `issuing_authority` is present.
2. There is no `ErrorItem` for data element `issuing_authority`.
