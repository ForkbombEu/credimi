# WS_RP_DM_IdentifyingData_Birthdate_PID_IETF-sd-jwt-vc_004

## Objective

This test case verifies that the claim `birthdate` contains a valid date. Note that `birthdate` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `birth_date`.

## References

- [PID rulebook] Annex 3.01 paragraph 3.1.4, Section 4.2, item 3 (Table 7)
- [ISO 8601-1:2019] 4.2.1, 4.3.2
- [RFC3339] Sections 5.6, 5.7

## Profile applicability

The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `birthdate` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify the value of date-fullyear.
2. Verify the value of date-month.
3. Verify the value of date-mday.

## Expected results

1. The value of date-fullyear is between "0000" and "9999" inclusive.
2. The value of date-month is between "01" and "12" inclusive.
3. The value of date-mday is between "01" and the maximum value specified in section 5.7 of RFC 3339 inclusive.
