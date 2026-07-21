# WS_RP_DM_IdentifyingData_Nationality_PID_ISO-mdoc_001

## Objective
This test case verifies that the data element `nationality` is present in the mdoc data. Note that `nationality` is the Attribute Identifier in ISO-mdoc for the Data Identifier nationality.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook, paragraph 3.1.2    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 1).

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in ISO-mdoc format with DocType = “eu.europa.ec.eudi.pid.1”

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved.

## Test Scenario
1. Verify the presence of a data element with identifier `nationality` in the device retrieval mdoc response. 
2. Verify the absence of an `ErrorItem` for data element `nationality`.

## Expected results
1. One data element with identifier `nationality` is present. 
2. There is no `ErrorItem` for data element `nationality`.