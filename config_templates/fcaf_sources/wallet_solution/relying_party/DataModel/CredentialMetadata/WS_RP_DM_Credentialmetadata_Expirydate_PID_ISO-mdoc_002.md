# WS_RP_DM_Credentialmetadata_Expirydate_PID_ISO-mdoc_002

## Objective

This test case verifies that the DataElementValue of data element `expiry_date` is encoded as a well-formed #6.0(tstr) or #6.1004(tstr). Note that `expiry_date` is the Attribute Identifier in ISO-mdoc for the Data Identifier `expiry_date`.

## References

- [PID rulebook] Annex 3.01, Section 4.1 (Table 6)
- [RFC7049] Sections 2.1, 2.4
- [RFC8610] Appendix D
- [RFC8943] Section 2.1

## Profile applicability

The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `expiry_date` is present in the mdoc data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.
4. The presence of data element `expiry_date` in the device retrieval mdoc response was verified.

## Test Scenario

1. Verify the major type encoded on the first byte of the CBOR data item.
2. Verify the value of the tag.
3. Verify the major type of the nested CBOR data item.
4. Verify the encoding of the data item value.

## Expected results

1. The major type value is equal to 6.
2. The value and encoding of the tag is one of the following: tag value = 0 (indicating a date-time tstr). The value of the additional information on the only byte is 0. Tag value = 1004 (indicating a full-date tstr). The value of the additional information on the first byte is 25 and the value of the next two bytes is '03 EC'.
3. The major type value is equal to 3.
4. All bytes in the data item value are UTF-8 encoded characters.
