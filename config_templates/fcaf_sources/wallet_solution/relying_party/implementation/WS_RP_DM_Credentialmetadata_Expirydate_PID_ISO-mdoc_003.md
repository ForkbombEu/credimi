# WS_RP_DM_Credentialmetadata_Expirydate_PID_ISO-mdoc_003

## Objective
This test case verifies that the format of the DataElementValue of data element `expiry_date` is correct. Note that `expiry_date` is the Attribute Identifier in ISO-mdoc for the Data Identifier expiry_date.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).  
RFC 3339 section 5.6

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `expiry_date` is present in the mdoc data.

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element `expiry_date` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the length of the data element. 
2. Verify the characters used in the format of the DataElementValue of the data element.

## Expected results
1. If the value of the tag is equal to:
* 0, the length of the data element is equal to 20 UTF-8 encoded characters.
* 1004, the length of the data element is equal to 10 UTF-8 encoded characters.
2. If the value of the tag is equal to:
* 0, the characters used have the following format: date-fullyear "-" date-month "-" date-mday "T" time-hour ":" time-minute ":" time-second time-offset.
* 1004, the characters used have the following format: date-fullyear "-" date-month "-" date-mday.
Where:  
* date-fullyear consists of four decimal digits.
* date-month, date-mday, time-hour, time-minute and time-second consist of two decimal digits.
* time-offset consists of one character: "Z".
