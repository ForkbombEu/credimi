# WS_RP_DM_AddressData_Emailaddress_PID_ISO-mdoc_001

## Objective

This test case verifies that the data element `email_address` is present in the mdoc data if this is indicated in the ICS. Note that `email_address` is the Attribute Identifier in ISO-mdoc for the Data Identifier `email_address`.

## References

- [PID rulebook] Annex 3.01, Section 4.1 (Table 2)

## Profile applicability

The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `email_address` is present in the mdoc data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.

## Test Scenario

1. Verify the presence of a data element with identifier `email_address` in the device retrieval mdoc response.
2. Verify the absence of an `ErrorItem` for data element `email_address`.

## Expected results

1. One data element with identifier `email_address` is present.
2. There is no `ErrorItem` for data element `email_address`.
