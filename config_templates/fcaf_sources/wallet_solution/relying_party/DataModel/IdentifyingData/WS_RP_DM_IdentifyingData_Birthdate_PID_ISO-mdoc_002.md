# WS_RP_DM_IdentifyingData_Birthdate_PID_ISO-mdoc_002

## Objective

This test case verifies that the DataElementValue of data element `birth_date` is encoded as a well-formed #6.1004(tstr). Note that `birth_date` is the Attribute Identifier in ISO-mdoc for the Data Identifier `birth_date`.

## References

- [PID rulebook] Annex 3.01 paragraph 3.1.4, Section 4.1 (Table 6)
- [RFC8943] Section 2.1

## Profile applicability

The EUDI wallet contains a Credential in ISO-mdoc format with DocType = “eu.europa.ec.eudi.pid.1”

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.
4. The presence of data element `birth_date` in the device retrieval mdoc response was verified.

## Test Scenario

1. Verify the major type encoded on the first byte of the CBOR data item.
2. Verify the value of the tag.
3. Verify the major type of the nested CBOR data item.
4. Verify the encoding of the data item value.

## Expected results

1. The major type value is equal to 6.
2. The tag value is equal to 1004 (indicating a full-date tstr). The value of the additional information on the first byte is 25 and the value of the next two bytes is '03 EC'.
3. The major type value is equal to 3.
4. All bytes in the data item value are UTF-8 encoded characters.
