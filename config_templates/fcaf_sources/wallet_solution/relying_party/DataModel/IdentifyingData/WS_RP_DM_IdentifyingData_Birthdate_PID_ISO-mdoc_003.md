# WS_RP_DM_IdentifyingData_Birthdate_PID_ISO-mdoc_003

## Objective

This test case verifies that the format of the DataElementValue of data element `birth_date` is correct. Note that `birth_date` is the Attribute Identifier in ISO-mdoc for the Data Identifier `birth_date`.

## References

- [PID rulebook] Annex 3.01 paragraph 3.1.4, Section 4.1 (Table 6)
- [RFC3339] Section 5.6

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

1. Verify the length of the data element.
2. Verify the characters used in the format of the DataElementValue of the data element.

## Expected results

1. The length of the data element is 10 UTF-8 encoded characters.
2. The characters used have the following format: date-fullyear "-" date-month "-" date-mday. Where: date-fullyear consists of four decimal digits. Date-month and date-mday consist of two decimal digits.
