# WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_ISO-mdoc_003

## Objective
This test case verifies that the DataElementValue of data element `issuing_jurisdiction` contains a valid country subdivision code. Note that `issuing_jurisdiction` is the Attribute Identifier in ISO-mdoc for the Data Identifier issuing_jurisdiction.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6). 
ISO 3166-2:2020, 6.1

## EUDI-wallet relevancy
EUDI_specific | EUDI_optional

## Profile applicability
The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `issuing_jurisdiction` is present in the EU PID data.

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence and format of data element `issuing_jurisdiction` in the device retrieval mdoc response was verified. 
E. The presence and format of data element `issuing_country` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the first two UTF-8 encoded characters in the data element. 
2. Verify the third UTF-8 encoded character in the data element. 
3. Verify that the remaining UTF-8 encoded characters in the data element constitute a valid value.

## Expected results
1. The first two UTF-8 encoded characters shall refer to the same country as the DataElementValue of data element `issuing_country`. 
2. The third UTF-8 encoded character is "-". 
3. The remaining UTF-8 encoded characters form a code corresponding to a country subdivision listed in ISO 3166-2:2020, 6.1 (according to the corresponding country code).