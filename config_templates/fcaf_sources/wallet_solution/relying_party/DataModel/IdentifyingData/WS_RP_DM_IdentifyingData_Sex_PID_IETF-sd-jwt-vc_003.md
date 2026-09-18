# WS_RP_DM_IdentifyingData_Sex_PID_IETF-sd-jwt-vc_003

## Objective

This test case verifies that the claim `sex` contains a valid value. Note that `sex` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier sex.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 2)

## Profile applicability

The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `sex` is included in a person identification data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `sex` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify that the claim `sex` contains a valid value for Sex.

## Expected results

1. The claim `sex` has one of the values specified in ISO/IEC 5218:2004, 2, i.e. 0, 1, 2 or 9 or one of the other values specified for Europe (i.e. 3, 4, 5, 6).
