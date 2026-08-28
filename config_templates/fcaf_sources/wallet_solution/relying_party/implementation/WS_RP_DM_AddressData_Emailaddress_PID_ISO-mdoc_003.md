# WS_RP_DM_AddressData_Emailaddress_PID_ISO-mdoc_003

## Objective
This test case verifies that the DataElementValue of data element email_address is an email address in conformance with [RFC 5322]. Note that `email_address` is the Attribute Identifier in ISO-mdoc for the Data Identifier email_address.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).  
RFC 5322 Section 3.4

## EUDI-wallet relevancy
EUDI_specific | EUDI_optional

## Profile applicability
The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element email_address is present in the mdoc data.

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element email_address in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the format of the CBOR data item. 
2. Verify the value of the email_address.

## Expected results
1. The data item contains at least 1 UTF-8 character. 
2. The electronic mail address of the user to whom the person identification data relates, is in conformance with [RFC 5322].