# WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003

## Objective

This test case verifies that the value of the claim `email` is an email address in conformance with [RFC 5322]. Note that `email` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier `email_address`.

## References

- [PID rulebook] Annex 3.01, Section 4.2 (Table 2)
- [RFC5322] Section 3.4

## Profile applicability

The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `email` is included in a person identification data.

## EUDI-wallet relevancy

EUDI_specific | EUDI_optional

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.
4. The presence of claim `email` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario

1. Verify the value of the email address.

## Expected results

1. The electronic mail address of the user to whom the person identification data relates, is in conformance with [RFC 5322].
