# WS_RP_DM_IdentifyingData_Nationality_PID_ISO-mdoc_003

## Objective

This test case verifies that the DataElementValue of data element `nationality` contains a valid country code. Note that `nationality` is the Attribute Identifier in ISO-mdoc for the Data Identifier nationality.

## References

- [PID rulebook] Annex 3.01 paragraph 3.1.2, Section 4.1 (Table 6)
- [ISO 3166-1:2020] 6.1
- [ISO 3166-1:2020] 8.3

## Profile applicability

The EUDI wallet contains a Credential in ISO-mdoc format with DocType = “eu.europa.ec.eudi.pid.1”

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.
4. The presence of data element `nationality` in the device retrieval mdoc response was verified.

## Test Scenario

1. Verify that the DataElementValue of the data element contains a valid country code.

## Expected results

1. The DataElementValue has two characters, an alpha-2 code, corresponding to a country name: Listed in ISO 3166-1:2020, 6.1. Not listed in ISO 3166-1:2020, 6.1. In this case, the user-assigned country codes may be used, which consist in the series of letters AA, QM to QZ, XA to XZ, and ZZ (ISO 3166-1:2020, 8.3).
