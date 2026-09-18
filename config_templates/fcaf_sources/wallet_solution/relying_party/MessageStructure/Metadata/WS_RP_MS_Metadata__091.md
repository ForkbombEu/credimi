# WS_RP_MS_Metadata_091

## Objective

Verify that the EUDI Wallet accepts a COSE-based Referenced Token containing a valid Status CBOR structure (Map) with at least a "status_list" entry.

## References

- [Token Status List] Section 6.3

## Profile applicability

The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in CWT format

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
2. The Referenced Token includes the Status element that contains the Status_list element (index and URI)
3. The EUDI Wallet has retrieved a Status List Token from the referenced URI
4. The Status List Token is validated
5. The Issuer has provided a COSE-based Referenced Token to the EUDI Wallet.
6. The Referenced Token contains a valid Status CBOR structure (Map) with at least a "status_list" entry.

## Test Scenario

1. Verify the presence and content of the Status CBOR structure.

## Expected results

1. The Status CBOR Map contains a valid "status_list" entry and the token is accepted.
