# WS_RP_DM_IdentifyingData_Sex_PID_IETF-sd-jwt-vc_001

## Objective

This test case verifies that the claim `sex` is present in the Credential in IETF SD-JWT VC format. Note that `sex` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier sex.

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

## Test Scenario

1. Verify the presence of a claim with identifier `sex` in the Credential presented to the Verifier in IETF SD-JWT VC format.

## Expected results

1. One claim with identifier `sex` is present.
