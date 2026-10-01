# WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_003

## Objective

This test case verifies that the format of the claim `date_of_issuance` is correct. Note that `date_of_issuance` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `issuance_date`.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 8)
- [RFC3339] Section 5.6

## Profile applicability

The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `date_of_issuance` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify the length of the claim `date_of_issuance`.
2. Verify the characters used in the format of the value of the claim.

## Expected results

1. The length of the claim is 10 UTF-8 encoded characters.
2. The characters used have the following format: date-fullyear "-" date-month "-" date-mday. Where:
    - date-fullyear consists of four decimal digits.
    - date-month and date-mday consist of two decimal digits.
