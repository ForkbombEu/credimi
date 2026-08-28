# WS_RP_SH_Encoding_TextualEncoding022

## Objective
Test that when the wallet sends the authorization response using HTTP POST, the names and values in the body MUST be encoded using UTF-8

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet has a credential containing non-ASCII characters

## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends an Authorization request to the wallet, with response mode=direct_post.jwt, requesting the non_ASCII credential
3. Wallet processes request
4. Test the HTTP POST body sent by wallet

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. Wallet processes request and performs HTTP POST to the response_uri.
4. All special characters (e.g. in names or values) are correctly encoded using UTF-8 bytes.