# WS_RP_DM_AddressData_Residentcountry_PID_ISO-mdoc_003

## Objective
This test case verifies that the DataElementValue of data element `resident_country` contains a valid country code. Note that `resident_country` is the Attribute Identifier in ISO-mdoc for the Data Identifier resident_country.


## References
European Digital Identity Wallet, ARF 2.0, Annex 3.01 pid rulebook    
"ANNEXES to the COMMISSION IMPLEMENTING REGULATION amending Implementing Regulation (EU) 2024/2977, (EU) 2024/2979, (EU) 2024/2980 and (EU) 2024/2982 as regards applicable standards and specifications and correcting Implementing Regulation (EU) 2024/2980" section 4.1 (Table 6). 
ISO 3166-1:2020, 6.1 ISO 3166-1:2020, 8.3

## EUDI-wallet relevancy
EUDI_specific | EUDI_optional

## Profile applicability
The EUDI wallet contains a Credential in ISO-mdoc format with DocType = "eu.europa.ec.eudi.pid.1". Data element `resident_country` is present in the mdoc data.

## Preconditions
A. A device retrieval mdoc request was sent to the EUDI wallet, to retrieve the document with DocType = "eu.europa.ec.eudi.pid.1". 
B. All mandatory data elements within namespace "eu.europa.ec.eudi.pid.1" and all data elements indicated as present in the ICS were requested. 
C. The device retrieval mdoc response was retrieved. 
D. The presence of data element `resident_country` in the device retrieval mdoc response was verified.

## Test Scenario
1. Verify that the DataElementValue of the data element contains a valid country code.

## Expected results
1. The DataElementValue has two characters, an alpha-2 code, corresponding to a country name: Listed in ISO 3166-1:2020, 6.1. Not listed in ISO 3166-1:2020, 6.1. In this case, the user-assigned country codes may be used, which consist in the series of letters AA, QM to QZ, XA to XZ, and ZZ (ISO 3166-1:2020, 8.3).