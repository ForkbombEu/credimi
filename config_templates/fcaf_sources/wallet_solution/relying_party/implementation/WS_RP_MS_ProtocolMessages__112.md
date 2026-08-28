# WS_RP_MS_ProtocolMessages_112

## Objective
Test the Wallet checks a claims object property "id" must consist of alphanumeric, underscore (_), or hyphen (-) characters. 

## References
[OID4VP] Sections 6.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The verifier sends a DCQL query containing a claims object with malformed "id" property whereby it contains a character NOT of the form alphanumeric, underscore (_), or hyphen (-).
3. Wallet handles Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet detects malformed "id" and returns an invalid_request error.

