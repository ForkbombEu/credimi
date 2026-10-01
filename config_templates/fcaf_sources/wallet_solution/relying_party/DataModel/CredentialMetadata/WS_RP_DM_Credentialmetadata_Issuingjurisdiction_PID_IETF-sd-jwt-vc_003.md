# WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_003

## Objective

This test case verifies that the value of the claim `issuing_jurisdiction` contains a valid country subdivision code. Note that `issuing_jurisdiction` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `issuing_jurisdiction`.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 5)
- [ISO 3166-2:2020] 8

## Profile applicability

The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `issuing_jurisdiction` is included in a person identification data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `issuing_jurisdiction` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify the first two UTF-8 encoded characters in claim `issuing_jurisdiction`.
2. Verify the third UTF-8 encoded character in the claim `issuing_jurisdiction`.
3. Verify that the remaining UTF-8 encoded characters in the claim `issuing_jurisdiction` constitute a valid value.

## Expected results

1. The first two UTF-8 encoded characters refer to the same country as the claim `issuing_country`.
2. The third UTF-8 encoded character is "-".
3. The remaining UTF-8 encoded characters form a code corresponding to a country subdivision listed in ISO 3166-2:2020, 6.1 (according to the corresponding country code).
