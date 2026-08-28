# WS_RP_DM_IdentifyingData_Portrait_PID_ISO-mdoc_003

## Objective
This test case verifies that the DataElementValue of data element `portrait` contains a valid encoded `portrait` image. Note that `portrait` is the Attribute Identifier in ISO-mdoc for the Data Identifier portrait.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6).

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in ISO-mdoc format with DocType = “eu.europa.ec.eudi.pid.1”

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element `portrait` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the first two bytes of the CBOR byte string. 
2. Verify that the encoded image format corresponds to the value of the first two bytes.

## Expected results
1. The first two bytes are equal to 'FF D8'. 
2. If the first two bytes of the CBOR byte string are equal to: 'FF D8', the byte string contains a JPEG encoded image.