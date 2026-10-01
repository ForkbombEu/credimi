# Testing ISO mdoc presentation

## Test specification approach for ISO/IEC 18013-5

The test specification approach for ISO/IEC 18013-5 (mdoc) slightly deviates from that for other STS. Given the test method coverage for 18013-5 is standardized in 18013-6, the FCAF follows the 18013-6 approach. Effectively, a profile for ISO/IEC TS 18013-6:2025 for EUDI-wallets is defined.

Below you will find a partially filled Implementation Conformity Statement (ICS) form template. Certain options have been pre-selected as they are applicable for any EUDI-wallet implementations. Remaining options are for the implementor to fill, matching their implementation choices.

## EUDI-wallet Implementation Conformance Statement template

The ICS below is pre-filled with mandatory items for EUDI-wallets, aligned with Annex B of ISO/IEC TS 18013-6:2025. All items are 'EUDI_generic' and the underlying requirements for implementations are specified in ISO/IEC 18013-5.

Test cases are specified in ISO/IEC TS 18013-6:2025. This test standards allows selection of applicable test cases using the statements below.

Note that data model and use case test cases are excluded here, as they will be covered under tests related to "the mDL rulebook". Testing of other doctypes, like PID, are likewise covered elsewhere.

Further note that IACA, linking and Document signer certificate test are applicable for the mDL issuing authority, or QEAA/PubEAA Provider in EUDI-wallet terminology. Reader authentication certificates tests are applicable for  Readers, or Relying Party (instances) and Providers of Wallet-Relying Party Access Certificates in EUDI-wallet terminology.

### Technologies test cases

The "Technologies" layer of ISO/IEC TS 18013-6 matches closest with the "Interactions" layer for EUDI-wallet, and "Message structure" in similar fashion. The various test areas of 18013-6 align with areas of EUDI-wallet FCAF, although not exactly 1-on-1. This mapping has not been further detailed.

| # | ICS | value | EUDI-wallet |
|:--- |:--- |:--- |:--- |
| 45 | mdoc supports device engagement using NFC Static Handover. | Yes/No | EUDI_optional. |
| 46 | mdoc supports device engagement using NFC Negotiated Handover. | Yes/No | EUDI_optional. |
| 47 | mdoc supports device engagement using QR code. | Yes/No | EUDI_optional. |
| 48 | mdoc supports device retrieval using NFC. | Yes/No | EUDI_optional. |
| 49 | mdoc supports extended-length APDUs for device retrieval using NFC. | Yes/No | EUDI_optional. |
| 50 | mdoc supports BLE version 4.2 (or above) and LE Data Packet Length Extension. | Yes/No | EUDI_optional. |
| 51 | mdoc supports device retrieval using BLE in mdoc central client mode. | Yes/No | EUDI_optional. |
| 52 | If BLE in mdoc central client mode is used for device retrieval, mdoc verifies the value of the Ident characteristic. | Yes/No | EUDI_optional. |
| 53 | mdoc supports the L2CAP transmission profile if it is acting as the GATT client for device retrieval using BLE. | Yes/No | EUDI_optional. |
| 54 | mdoc supports device retrieval using BLE in mdoc peripheral server mode. | Yes/No | EUDI_optional. |
| 55 | mdoc supports the L2CAP transmission profile if it is acting as the GATT server for device retrieval using BLE. | Yes/No | EUDI_optional. |
| 56 | mdoc supports device retrieval using Wi-Fi Aware. | Yes/No | EUDI_optional. |
| 57 | mdoc supports the NCS-PK-2WDH-128 cipher suite for Wi-Fi Aware. | Yes/No | EUDI_optional. |
| 58 | mdoc supports server retrieval using OIDC. | ~~Yes~~/No | EUDI_prohibited; Server retrieval is explicitly prohibited in [CIR 2024/2979 amended]. |
| 59 | mdoc supports server retrieval using WebAPI. | ~~Yes~~/No | EUDI_prohibited; Server retrieval is explicitly prohibited in [CIR 2024/2979 amended]. |
| 60 | mdoc supports transferring server retrieval information in the device engagement structure. | ~~Yes~~/No | EUDI_prohibited; Server retrieval is explicitly prohibited in [CIR 2024/2979 amended]. |
| 61 | mdoc implements a time-out for the time between sending device engagement data and receiving the session establishment message when using QR code for device engagement. | Yes/No | EUDI_optional |
| 62 | If yes, how many seconds is the time-out period implemented by the mdoc?	 | | EUDI_optional |
| 63 | mdoc implements a time-out for the time between sending device engagement data and receiving the session establishment message when using NFC for device engagement. | Yes/No| EUDI_optional |
| 64 | If yes, how many seconds is the time-out period implemented by the mdoc?	 | | EUDI_optional |

### Security mechanisms test cases

The "Security mechanisms" test layer of ISO/IEC 18013-6 matches the EUDI-wallet FCAF "Security mechanisms" test layer. Test various test areas of 18013-6 align with areas of EUDI-wallet FCAF, although not exactly 1-on-1. This mapping has not been further detailed.

| # | ICS | value | EUDI-wallet |
|:--- |:--- |:--- |:--- |
| 65 | Which curves does the mdoc support for session establishment? Select all that are supported. | Curve P-256 | EUDI_optional |
| | | Curve P-384 | EUDI_optional |
| | | Curve P-521 | EUDI_optional |
| | | ~~X25519~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | ~~X448~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP256r1 | EUDI_optional |
| | | ~~brainpoolP320r1~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP384r1 | EUDI_optional |
| | | brainpoolP512r1 | EUDI_optional |
| 66 | mdoc supports exchanging more than one device retrieval mdoc request and response with the mdoc reader in a single session. | Yes/No | EUDI_optional |
| 67 | If yes, how many seconds is the time-out period for session termination implemented by the mdoc? | | EUDI_optional |
| 68 | Which curves does the issuing authority support for issuer data authentication on this mdoc? Select all that are supported. | Curve P-256 | EUDI_optional |
| | | Curve P-384 | EUDI_optional |
| | | Curve P-521 | EUDI_optional |
| | | ~~X25519~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | ~~X448~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP256r1 | EUDI_optional |
| | | ~~brainpoolP320r1~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP384r1 | EUDI_optional |
| | | brainpoolP512r1 | EUDI_optional |
| 69 | The mdoc supports mdoc MAC authentication. | Yes/No | EUDI_optional (note: [ETSI 119 472-1] v1.1.1 excludes use of MAC authentication, v1.1.4 allows MAC authentication) |
| 70 | If yes, which curves does the mdoc support for mdoc MAC authentication? Select all that are supported.| Curve P-256 | EUDI_optional |
| | | Curve P-384 | EUDI_optional |
| | | Curve P-521 | EUDI_optional |
| | | ~~X25519~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | ~~X448~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP256r1 | EUDI_optional |
| | | ~~brainpoolP320r1~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP384r1 | EUDI_optional |
| | | brainpoolP512r1 | EUDI_optional |
| 71 | The mdoc supports mdoc ECDSA/~~EdDSA~~ authentication. | Yes/No | |
| 72 | If yes, which curves does the mdoc support for mdoc ECDSA/EdDSA authentication? Select all that are supported. | Curve P-256 | EUDI_optional |
| | | Curve P-384 | EUDI_optional |
| | | Curve P-521 | EUDI_optional |
| | | ~~X25519~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | ~~X448~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP256r1 | EUDI_optional |
| | | ~~brainpoolP320r1~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP384r1 | EUDI_optional |
| | | brainpoolP512r1 | EUDI_optional |
| 73 | mdoc supports mdoc reader authentication. | Yes/~~No~~ | EUDI_required |
| 74 | If yes, which curves does the mdoc support for mdoc reader authentication? Select all that are supported. | Curve P-256 | EUDI_optional |
| | | Curve P-384 | EUDI_optional |
| | | Curve P-521 | EUDI_optional |
| | | ~~X25519~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | ~~X448~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP256r1 | EUDI_optional |
| | | ~~brainpoolP320r1~~ | EUDI_prohibited; algorithm not listed in [EUCC] |
| | | brainpoolP384r1 | EUDI_optional |
| | | brainpoolP512r1 | EUDI_optional |
| 75 | If yes, if mdoc reader authentication fails, does the mdoc notify the mdoc holder that the mdoc verifier’s identity could not be verified? | Yes/ No | EUDI_optional. |
| 76 | If yes, if mdoc reader authentication fails, are there any data elements that the mdoc will not release? If so, please list them all by namespace and identifier.|  Yes / No | |
| 77 | If yes, mdoc supports retrieving OCSP information, if available, when verifying a mdoc reader authentication certificate. | Yes / No | |
| 78 | A test CRL for all IACA root certificate provided by applicant is available during testing. | Yes / No | |
| 79 | A test CRL for all Document Signer certificate provided by applicant is available during testing. | Yes / No | |
| 80 | If yes, mdoc supports retrieving CRL information when verifying a mdoc reader authentication certificate.	| Yes / No | |
