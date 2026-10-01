# WS_RP_DM_IdentifyingData_Birthplace_PID_ISO-mdoc_004

## Objective

This test case verifies that the value of each data element present in `place_of_birth` is a well-formed CBOR text string. Note that `place_of_birth` is the Attribute Identifier in ISO-mdoc for the Data Identifier birth_place.

## References

- [PID rulebook] Annex 3.01 paragraph 3.1.5, Section 4.1 (Table 6)
- [RFC7049] Section 2.1

## Profile applicability

The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1".

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1".
2. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested.
3. The device retrieval mdoc response was retrieved.
4. The presence of data element `place_of_birth` in the device retrieval mdoc response was verified.

## Test Scenario

For each data item present, perform the following check for the value of the key/value pair:
1. Verify the encoding of the data item value.

## Expected results

1. The major type value is equal to 3.
