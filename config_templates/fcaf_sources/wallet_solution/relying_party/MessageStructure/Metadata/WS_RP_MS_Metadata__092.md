# WS_RP_MS_Metadata_092

## Objective

Verify that the EUDI Wallet rejects a COSE-based Referenced Token containing an empty Status CBOR Map.

## References

- [Token Status List] Section 6.3

## Profile applicability

The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
2. The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
3. The Referenced Token contains an empty Status CBOR Map.

## Test Scenario

1. Verify the Wallet's handling of the empty Status CBOR Map.

## Expected results

1. The Referenced Token is rejected.
