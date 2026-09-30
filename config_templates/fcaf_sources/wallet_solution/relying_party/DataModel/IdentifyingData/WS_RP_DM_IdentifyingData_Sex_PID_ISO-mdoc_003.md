# WS_RP_DM_IdentifyingData_Sex_PID_ISO-mdoc_003

## Objective

This test case verifies that the DataElementValue of data element `sex` contains a valid value.  Note that `sex` is the Attribute Identifier in ISO-mdoc for the Data Identifier sex.

## References

- [PID rulebook] Annex 3.01, Section 4.1 (Table 6)
- [ISO/IEC 5218:2004] 2

## Profile applicability

The EUDI wallet contains a Credential in ISO-mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `sex` is present in the mdoc data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.
4. The presence of data element `sex` in the device retrieval mdoc response was verified.

## Test Scenario

1. Verify that the DataElementValue of the data element contains a valid value for Sex.

## Expected results

1. The DataElementValue has one of the values specified in ISO/IEC 5218:2004, 2, i.e. 0, 1, 2 or 9 or one of the other values specified for Europe (i.e. 3, 4, 5, 6).
