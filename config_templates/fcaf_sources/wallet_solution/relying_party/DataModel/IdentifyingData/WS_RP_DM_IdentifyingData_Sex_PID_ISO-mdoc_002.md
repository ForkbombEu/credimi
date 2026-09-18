# WS_RP_DM_IdentifyingData_Sex_PID_ISO-mdoc_002

## Objective

This test case verifies that the DataElementValue of data element `sex` is a well-formed CBOR unsigned integer.  Note that `sex` is the Attribute Identifier in ISO-mdoc for the Data Identifier sex.

## References

- [PID rulebook] Annex 3.01, Section 4.1 (Table 6)
- [RFC7049] Section 2.1

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

1. Verify the major type encoded on the first byte of the CBOR data item.

## Expected results

1. The major type value is equal to 0.
