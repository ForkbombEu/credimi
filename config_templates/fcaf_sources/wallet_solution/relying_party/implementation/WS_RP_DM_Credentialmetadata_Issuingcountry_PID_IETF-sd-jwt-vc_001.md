# WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_001

## Objective
This test case verifies that the claim `issuing_country` is present in the Credential in IETF SD-JWT VC format. Note that `issuing_country` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier issuing_country. 

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook   
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.2 (Table 5).

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## Preconditions
A. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.  
B. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested. 
C. EUDI wallet presented the Credential successfully.

## Test Scenario
1. Verify the presence of a claim with identifier `issuing_country` in the Credential presented to the Verifier in IETF SD-JWT VC format.

## Expected results
1. One claim with identifier `issuing_country` is present.