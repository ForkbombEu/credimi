# WS_RP_DM_IdentifyingData_Portrait_PID_IETF-sd-jwt-vc_003

## Objective
This test case verifies that the claim `picture` is a data URL containing a base64-encoded `picture` in JPEG format. Note that `picture` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier portrait.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook   
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.2 (Table 7).

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## Preconditions
A. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.  
B. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested. 
C. EUDI wallet presented the Credential successfully. 
D. The presence of claim `picture` in the IETF SD-JWT VC Credential presented was verified. 
E. It was verified that the claim `picture` is a String.

## Test Scenario
1. Verify that claim `picture` is a data URL containing a base64-encoded `picture` in JPEG format.

## Expected results
1. The claim `picture` is a data URL containing a base64-encoded `picture` in JPEG format.