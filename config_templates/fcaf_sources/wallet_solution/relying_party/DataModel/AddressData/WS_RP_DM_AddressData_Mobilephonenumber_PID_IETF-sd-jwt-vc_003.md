# WS_RP_DM_AddressData_Mobilephonenumber_PID_IETF-sd-jwt-vc_003

## Objective

This test case verifies that the value of the claim `phone_number` is formatted as an international phone number. Note that `phone_number` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `mobile_phone_number`.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 2)

## Profile applicability

The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `phone_number` is included in a person identification data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `phone_number` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify the length of the claim `phone_number`.
2. Verify the value from `phone_number`.

## Expected results

1. The length of the claim `phone_number` is at least 8 UTF-8 characters.
2. The `phone_number` value is the mobile phone number of the user to whom the person identification data relates, starting with the '+' symbol as the international code prefix and the country code, followed by numbers only.
