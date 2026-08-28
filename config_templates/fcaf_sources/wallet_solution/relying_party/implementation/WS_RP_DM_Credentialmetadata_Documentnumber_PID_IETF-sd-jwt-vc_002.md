# WS_RP_DM_Credentialmetadata_Documentnumber_PID_IETF-sd-jwt-vc_002

## Objective
This test case verifies that the value of the `document_number` is a String. Note that `document_number` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier document_number.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook. 
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.2 (Table 8).

## EUDI-wallet relevancy
EUDI_specific | EUDI_optional

## Profile applicability
The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".
  
## Preconditions
A. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.  
B. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested. 
C. EUDI wallet presented the Credential successfully.
D. The presence of claim `document_number` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario
1. Verify that the claim `document_number` is a String encoded in UTF-8, supporting the full Unicode range.

## Expected results
1. The claim `document_number` is a String encoded in UTF-8, supporting the full Unicode range.