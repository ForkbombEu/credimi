# WS_RP_DM_IdentifyingData_Birthplace_PID_ISO-mdoc_005

## Objective

This test case verifies that the country item of data element `place_of_birth` is a valid country code. Note that `place_of_birth` is the Attribute Identifier in ISO-mdoc for the Data Identifier birth_place.

## References

- [PID rulebook] Annex 3.01 paragraph 3.1.5, Section 4.1 (Table 6)
- [ISO/IEC 3166-1]

## Profile applicability

The mdoc contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1".

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.
4. The presence of the item ‘country’ in data element `place_of_birth` in the device retrieval mdoc response was verified.

## Test Scenario

1. Verify that the value of the data element `country` in map `place_of_birth` contains a valid country code.

## Expected results

1. The DataElementValue has two characters, an alpha-2 code, corresponding to a country name: Listed in ISO 3166-1:2020, 6.1. Not listed in ISO 3166-1:2020, 6.1. In this case, the user-assigned country codes may be used, which consist in the series of letters AA, QM to QZ, XA to XZ, and ZZ (ISO 3166-1:2020, 8.3).
