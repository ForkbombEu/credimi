# WS_RP_DM_IdentifyingData_Birthdate_PID_IETF-sd-jwt-vc_003

## Objective
This test case verifies that the format of the claim `birthdate` is correct. Note that `birthdate` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier birth_date.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook, paragraph 3.1.4   
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.2 (Table 7).RFC 3339 section 5.6

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## Preconditions
A. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.  
B. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested. 
C. EUDI wallet presented the Credential successfully. 
D. The presence of claim `birthdate` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario
1. Verify the length of the claim `birthdate`.
2. Verify the characters used in the format of the value of the claim.

## Expected results
1. The length of the claim is 10 UTF-8 encoded characters. 
2. The characters used have the following format: date-fullyear "-" date-month "-" date-mday. Where: 
* date-fullyear consists of four decimal digits. 
* Date-month and Date-mday consist of two decimal digits.