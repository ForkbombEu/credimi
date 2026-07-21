# WS_RP_DM_Credentialmetadata_Domestic_PID_ISO-mdoc_001

## Objective
This test case verifies domestic elements in an appropriate namespace are present, if these have been indicated in the ICS.

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

## Test Scenario
1. Verify data elements in the domestic namespace are present.

## Expected results
1. One or more data elements in the indicated namespace are present.