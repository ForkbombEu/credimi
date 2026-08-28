# WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_004

## Objective
This test case verifies that the claim `date_of_expiry` contains a valid date. Note that `date_of_expiry` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier expiry_date. 

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook   
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.2 item 4 (Table 8). ISO 8601-1:2019, 4.2.1, 4.3.2  
 RFC 3339 section 5.6, 5.7

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1".

## Preconditions
A. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.  
B. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested. 
C. EUDI wallet presented the Credential successfully. 
D. The presence of claim `date_of_expiry` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario
1. Verify the value of date-fullyear. 
2. Verify the value of date-month. 
3. Verify the value of date-mday.

## Expected results
1. The value of date-fullyear is between "0000" and "9999" inclusive. 
2. The value of date-month is between "01" and "12" inclusive. 
3. The value of date-mday is between "01" and the maximum value specified in section 5.7 of RFC 3339 inclusive.