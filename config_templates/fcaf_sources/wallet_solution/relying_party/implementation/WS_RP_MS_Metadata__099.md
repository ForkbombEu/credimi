# WS_RP_MS_Metadata_099

## Objective
Verify that the EUDI Wallet rejects a COSE-based Referenced Token where the "uri" field within "status_list" is malformed.

## References
[Token Status List (TSL) draft-20] Section 6.3

## Profile applicability
The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy
EUDI_generic / EUDI_required

## Preconditions
A) The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
B) The Referenced Token includes the Status element that contains the Status_list element (index and URI)
C) The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
D) The Referenced Token contains the "uri" field within "status_list" set to a malformed URI.

## Test Scenario
1. Provide a Referenced Token where the "uri" value has no scheme (e.g., example.com/statuslists/1). Verify the Wallet's handling.
2. Provide a Referenced Token where the "uri" value contains an unencoded space in the host (e.g., https://exam ple.com/statuslists/1). Verify the Wallet's handling.
3. Provide a Referenced Token where the "uri" value contains invalid percent-encoding (e.g., https://example.com/statuslists/%GG). Verify the Wallet's handling.
4. Provide a Referenced Token where the "uri" value contains illegal characters (e.g., https://example.com/statuslists/<token>|1). Verify the Wallet's handling.
5. Provide a Referenced Token where the "uri" value is an empty string. Verify the Wallet's handling.

## Expected results
1. The Wallet rejects the Referenced Token (missing scheme).
2. The Wallet rejects the Referenced Token (invalid character in host).
3. The Wallet rejects the Referenced Token (malformed percent-encoding).
4. The Wallet rejects the Referenced Token (illegal characters <, >, | are not permitted in URIs per RFC 3986).
5. The Wallet rejects the Referenced Token (empty URI).

