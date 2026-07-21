# WS_RP_DM_IdentifyingData_Birthplace_PID_ISO-mdoc_003

## Objective
This test case verifies that the correct key-value pairs are present in each data element present in `place_of_birth`. Note that `place_of_birth` is the Attribute Identifier in ISO-mdoc for the Data Identifier birth_place.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook, paragraph 3.1.5    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).  
RFC 7049 section 2.1

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1".

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element `place_of_birth` in the device retrieval mdoc response was verified.

## Test Scenario
For each item present, perform the following check: 
1. Verify the value of the key for each nested key/value pair. 
2. Verify that each key is unique within the data item.

## Expected results
1. The data element item contains between 1 and 3 pairs inclusive, with the following value(s) for the key/ value pair(s): — "country" (optional), — "region" (optional), — "locality" (optional). 
2. Each key is unique within the data item.