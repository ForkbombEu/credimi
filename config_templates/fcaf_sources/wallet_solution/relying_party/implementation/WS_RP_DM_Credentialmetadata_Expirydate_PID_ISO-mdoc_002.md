# WS_RP_DM_Credentialmetadata_Expirydate_PID_ISO-mdoc_002

## Objective
This test case verifies that the DataElementValue of data element `expiry_date` is encoded as a well-formed #6.0(tstr) or #6.1004(tstr). Note that `expiry_date` is the Attribute Identifier in ISO-mdoc for the Data Identifier expiry_date.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).  
RFC 7049 section 2.1, 2.4  
RFC 8610 Appendix D  
RFC 8943 section 2.1

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
1. Verify the major type encoded on the first byte of the CBOR data item. 
2. Verify the value of the tag. 
3. Verify the major type of the nested CBOR data item. 
4. Verify the encoding of the data item value.

## Expected results
1. The major type value is equal to 6. 
2. The value and encoding of the tag is one of the following: tag value = 0 (indicating a date-time tstr). The value of the additional information on the only byte is 0. Tag value = 1004 (indicating a full-date tstr). The value of the additional information on the first byte is 25 and the value of the next two bytes is '03 EC'. 
3. The major type value is equal to 3. 
4. All bytes in the data item value are UTF-8 encoded characters.