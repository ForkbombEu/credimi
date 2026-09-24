# WS_RP_DM_Credentialmetadata_Issuingauthority_PID_IETF-sd-jwt-vc_002

## Objective

This test case verifies that the claim `issuing_authority` is a String encoded in UTF-8. Note that `issuing_authority` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `issuing_authority`.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 8)

## Profile applicability

The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `issuing_authority` is included in a person identification data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `issuing_authority` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify that the claim `issuing_authority` is a String encoded in UTF-8, supporting the full Unicode range

## Expected results

1. The claim `issuing_authority` is a String encoded in UTF-8, supporting the full Unicode range.
