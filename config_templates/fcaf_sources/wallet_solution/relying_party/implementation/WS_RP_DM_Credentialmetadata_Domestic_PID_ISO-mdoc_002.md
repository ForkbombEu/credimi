# WS_RP_DM_Credentialmetadata_Domestic_PID_ISO-mdoc_002

## Objective
This test case verifies domestic elements are located in a valid domestic namespace, if these have been indicated in the ICS.

## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1.

## EUDI-wallet relevancy
EUDI_specific | EUDI_required

## Profile applicability
The EUDI wallet contains a Credential in mdoc format with DocType = "eu.europa.ec.eudi.pid.1", which contains data elements in a domestic namespace.

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested, as well as data elements in the domestic namespace indicated in the ICS. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element in the domestic namespace in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify the data elements in the domestic namespace comply use a valid the domestic namespace identifier.

## Expected results
1. The domestic namespace: Use a prefix ‘eu.europa.ec.eudi.pid.’ Followed by a ISO 3166-1 alpha-2 country code or the ISO 3166-2 region code. Optionally followed by a dot and a number.