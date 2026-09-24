# WS_RP_DM_IdentifyingData_Portrait_PID_IETF-sd-jwt-vc_002

## Objective

This test case verifies that the claim `picture` is a String. Note that `picture` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier portrait.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 7)

## Profile applicability

The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `picture` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify that the claim `picture` is a String.

## Expected results

1. The claim `picture` is a String.
