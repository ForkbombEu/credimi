# WS_RP_DM_AddressData_Mobilephonenumber_PID_ISO-mdoc_003

## Objective
This test case verifies that the DataElementValue of data element  `mobile_phone_number` is formatted as an international phone number. Note that  `mobile_phone_number` is the Attribute Identifier in ISO-mdoc for the Data Identifier mobile_phone_number.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).  

## EUDI-wallet relevancy
EUDI_specific | EUDI_optional

## Profile applicability
The EUDI wallet contains a Credential in ISO-mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element  `mobile_phone_number` is present in the mdoc data.

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element `mobile_phone_number` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the format of the CBOR data item. 
2. Verify the value of the Mobile Phone Number.  


## Expected results
1. The data item contains at least 8 UTF-8 characters. 
2. Mobile telephone number of the User to whom the person identification data relates, starting with the '+' symbol as the international code prefix and the country code, followed by numbers only. 
