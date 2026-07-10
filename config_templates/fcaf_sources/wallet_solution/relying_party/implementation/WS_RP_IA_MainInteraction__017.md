# WS_RP_IA_MainInteraction_017

## Objective
Test that when claims is absent in the request the Wallet MUST return only the claims that are mandatory to present, NO selectively disclosable claims.

## References
[OID4VP 6.4]

## Profile applicability
ISO mDL

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet holds an ISO mDL with elements: family_name, document_number, and portrait.

## Test Scenario
1. The Wallet engages with the Verifier
2. A Verifier sends a request for a Credential the wallet holds, but the claims object is absent.
3. Wallet handles request

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet received query, and generates a valid response showing licence is authentic but giving zero personal data