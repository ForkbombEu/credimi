# WS_RP_DM_IdentifyingData_Givenname_PID_IETF-sd-jwt-vc_002

## Objective

This test case verifies that the claim `given_name` is a String encoded in UTF-8. Note that `given_name` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `given_name`.

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
4. The presence of claim `given_name` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify that the claim `given_name` is a String encoded in UTF-8, supporting the full Unicode range.

## Expected results

1. The claim `given_name` is a String encoded in UTF-8, supporting the full Unicode range.
