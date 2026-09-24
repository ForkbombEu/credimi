# WS_RP_DM_IdentifyingData_Nationality_PID_IETF-sd-jwt-vc_001

## Objective

This test case verifies that the claim `nationalities` is present in the Credential in IETF SD-JWT VC format. Note that `nationalities` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier nationality.

## References

- [PID rulebook] Annex 3.01 paragraph 3.1.2, Section 4.2 (Table 1)

## Profile applicability

The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.

## Test Scenario

1. Verify the presence of a claim with identifier `nationalities` in the Credential presented to the Verifier in IETF SD-JWT VC format.

## Expected results

1. One claim with identifier `nationalities` is present.
