# WS_RP_DM_IdentifyingData_Birthplace_PID_ISO-mdoc_007

## Objective
This test case verifies that the region item of data element `place_of_birth` has only UTF-8 characters, and maximum length of 150 characters. Note that `place_of_birth` is the Attribute Identifier in ISO-mdoc for the Data Identifier birth_place.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook..    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).  
ISO/IEC 3166-1

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The mdoc contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1".

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of the item ‘locality’ in data element `place_of_birth` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify that the value of the data element `locality` in map `place_of_birth`.
2. Check the length of the `locality` value in map `place_of_birth`.

## Expected results
1. Value includes only UTF-8 characters. 
2. Length is equal or smaller than 150 characters.