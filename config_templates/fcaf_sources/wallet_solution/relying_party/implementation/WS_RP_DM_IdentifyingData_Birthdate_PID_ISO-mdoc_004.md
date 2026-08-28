# WS_RP_DM_IdentifyingData_Birthdate_PID_ISO-mdoc_004

## Objective
This test case verifies that the DataElementValue of data element `birth_date` contains a valid date. Note that `birth_date` is the Attribute Identifier in ISO-mdoc for the Data Identifier birth_date.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook, paragraph 3.1.4    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6). ISO 8601-1:2019, 4.2.1, 4.3.2  
RFC 3339 section 5.6, 5.7

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in ISO-mdoc format with DocType = “eu.europa.ec.eudi.pid.1”

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element `birth_date` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the value of date-fullyear. 
2. Verify the value of date-month. 
3. Verify the value of date-mday.

## Expected results
1. The value of date-fullyear is between "0000" and "9999" inclusive. 
2. The value of date-month is between "01" and "12" inclusive. 
3. The value of date-mday is between "01" and the maximum value specified in section 5.7 of RFC 3339 inclusive.