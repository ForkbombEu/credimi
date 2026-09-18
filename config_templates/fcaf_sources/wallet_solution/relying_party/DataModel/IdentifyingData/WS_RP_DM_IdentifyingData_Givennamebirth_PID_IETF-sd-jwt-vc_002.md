# WS_RP_DM_IdentifyingData_Givenname_PID_IETF-sd-jwt-vc_004

## Objective

This test case verifies that the claim `birth_given_name` is a String encoded in UTF-8. Note that `birth_given_name` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `given_name_birth`.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 7)

## Profile applicability

The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `birth_given_name` is included in a person identification data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `birth_given_name` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify that the claim `birth_given_name` is a String encoded in UTF-8, supporting the full Unicode range.

## Expected results

1. The claim `birth_given_name` is a String encoded in UTF-8, supporting the full Unicode range.
