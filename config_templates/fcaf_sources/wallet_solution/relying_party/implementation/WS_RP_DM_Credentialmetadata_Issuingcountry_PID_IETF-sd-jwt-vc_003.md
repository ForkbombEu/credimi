# WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_003

## Objective
This test case verifies that the value of the claim `issuing_country` contains a valid country code. Note that `issuing_country` is the Attribute Identifier in IETF SD-JWT VC for the Data Identifier issuing_country. 

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook   
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.2 (Table 5). 
ISO 3166-1:2020, 6.1 ISO 3166-1:2020, 8.3

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in SD-JWT VC format with vct = "urn:eudi:pid:". The claim `issuing_country` is included in a person identification data.

## Preconditions
A. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.  
B. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested. 
C. EUDI wallet presented the Credential successfully. 
D. The presence of claim `issuing_country` in the IETF SD-JWT VC Credential presented was verified.

## Test Scenario
1. Verify that the value of the claim `issuing_country` contains a valid country code.

## Expected results
1. The value of the claim `issuing_country`, an alpha-2 code, corresponding to a country name: Listed in ISO 3166-1:2020, 6.1. Not listed in ISO 3166-1:2020, 6.1. In this case, the user-assigned country codes may be used, which consist in the series of letters AA, QM to QZ, XA to XZ, and ZZ (ISO 3166-1:2020, 8.3).
