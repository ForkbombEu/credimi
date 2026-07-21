# WS_RP_MS_Metadata_103

## Objective
Verify that the number, keys, and value types of data items in the StatusListInfo CBOR structure within a COSE-based Referenced Token are correct.

## References
[Token Status List (TSL) draft-20] Section 6.3

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
B) The Referenced Token contains a status claim with a "status_list" CBOR structure.
C) The status_list structure passed basic CBOR well-formedness checks.

## Test Scenario
1. Verify the additional information encoded on the last five bits of the first byte of the status_list map.
2. Verify that the required key-value pairs are present with the correct CBOR types.
3. Verify that no unrecognized key-value pairs with key major type 3 (tstr) are present, other than "idx", "uri", and optionally "certificate".

## Expected results
1. The value of the additional information (encoding the number of key-value pairs in the map) is 2 or 3:
   - 2 if only the required fields are present (idx, uri)
   - 3 if the optional certificate field is also present
2. The key-value pairs present have the following properties:
   key major type = 3 (tstr), key value = "idx" & value major type = 0 (uint),
   key major type = 3 (tstr), key value = "uri" & value major type = 3 (tstr),
   optionally: key major type = 3 (tstr), key value = "certificate" & value major type = 2 (bstr).
3. No unrecognized key-value pairs with key major type 3 (tstr) are present beyond those listed in step 2.

