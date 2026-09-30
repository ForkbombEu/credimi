# WS_RP_DM_IdentifyingData_Birthplace_PID_IETF-sd-jwt-vc_002

## Objective

This test case verifies that the claim `place_of_birth` is encoded as a JSON structure. Note that `place_of_birth` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier birth_place.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 7)
- [RFC7049] Section 2.1

## Profile applicability

The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `place_of_birth` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify that claim is encoded as a JSON structure.

## Expected results

1. The claim is encoded as a JSON structure.
