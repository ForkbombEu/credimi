# WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003

## Objective
This test case verifies that the value of the claim `email` is an email address in conformance with [RFC 5322]. Note that `email` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier email_address.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook   
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.2 (Table 2).  
RFC 5322 Section 3.4

## EUDI-wallet relevancy
EUDI_specific | EUDI_optional

## Profile applicability
The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `email` is included in a person identification data.

## Preconditions
A. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.  
B. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested. 
C. EUDI wallet presented the Credential successfully. 
D. The presence of claim `email` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario
1. Verify the value of the email address.

## Expected results
1. The electronic mail address of the user to whom the person identification data relates, is in conformance with [RFC 5322].
