# WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_003

## Objective

This test case verifies that the value of the claim `issuing_country` contains a valid country code. Note that `issuing_country` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `issuing_country`.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 5)
- [ISO 3166-1:2020] 6.1
- [ISO 3166-1:2020] 8.3

## Profile applicability

The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `issuing_country` is included in a person identification data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `issuing_country` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify that the value of the claim `issuing_country` contains a valid country code.

## Expected results

1. The value of the claim `issuing_country`, an alpha-2 code, corresponding to a country name: Listed in ISO 3166-1:2020, 6.1. Not listed in ISO 3166-1:2020, 6.1. In this case, the user-assigned country codes may be used, which consist in the series of letters AA, QM to QZ, XA to XZ, and ZZ (ISO 3166-1:2020, 8.3).
