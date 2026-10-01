# WS_RP_MS_CredentialFormats_031

## Objective

Verify that the EUDI Wallet can receive, hold and store an SD-JWT VC Referenced Token that includes a status_list claim.

## References

- [Token Status List] Section 6.2

## Profile applicability

The Wallet supports revocation checking via the Token Status List mechanism; The Wallet supports Status List Tokens in JWT format

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. The EUDI Wallet requests and receives a valid Referenced Token issued by an Issuer
2. The Referenced Token includes the Status element that contains the Status_list element (index and URI)
3. The EUDI Wallet has retrieved a Status List Token from the referenced URI
4. The Status List Token is validated
5. The Issuer issues a SD-JWT VC Referenced Token with a status_list claim to the EUDI Wallet.

## Test Scenario

1. Verify that the Wallet accepts and stores the SD-JWT VC Referenced Token.

## Expected results

1. The SD-JWT VC Referenced Token with status_list claim is stored by the Wallet.
