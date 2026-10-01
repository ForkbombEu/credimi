# WS_RP_DM_AddressData_Mobilephonenumber_PID_ISO-mdoc_003

## Objective

This test case verifies that the DataElementValue of data element  `mobile_phone_number` is formatted as an international phone number. Note that  `mobile_phone_number` is the Attribute Identifier in ISO-mdoc for the Data Identifier `mobile_phone_number`.

## References

- [PID rulebook] Annex 3.01, Section 4.1 (Table 6)

## Profile applicability

The EUDI wallet contains a Credential in ISO-mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element  `mobile_phone_number` is present in the mdoc data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.
4. The presence of data element `mobile_phone_number` in the device retrieval mdoc response was verified.

## Test Scenario

1. Verify the format of the CBOR data item.
2. Verify the value of the Mobile Phone Number.

## Expected results

1. The data item contains at least 8 UTF-8 characters.
2. Mobile telephone number of the User to whom the person identification data relates, starting with the '+' symbol as the international code prefix and the country code, followed by numbers only.
