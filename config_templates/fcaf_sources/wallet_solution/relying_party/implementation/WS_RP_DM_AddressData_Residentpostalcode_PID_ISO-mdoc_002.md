# WS_RP_DM_AddressData_Residentpostalcode_PID_ISO-mdoc_002

## Objective
This test case verifies that the DataElementValue of data element `resident_postal_code` is a well-formed CBOR text string. Note that `resident_postal_code` is the Attribute Identifier in ISO-mdoc for the Data Identifier resident_postal_code.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook.    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).  
RFC 7049 section 2.1

## EUDI-wallet relevancy
EUDI_specific | EUDI_optional

## Profile applicability
The EUDI wallet contains a Credential in ISO-mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `resident_postal_code` is present in the mdoc data.

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element `resident_postal_code` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the major type encoded on the first byte of the CBOR data item. 
2. Verify the encoding of the data item value.

## Expected results
1. The major type value is equal to 3. 
2. All bytes in the data item value are UTF-8 encoded characters.