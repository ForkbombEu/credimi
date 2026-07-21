# WS_RP_IA_MainInteraction_030

## Objective
Test the Wallet returns presentations of a set of Credentials that match to one of the options inside the Credential Set Query.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query, including a "credential_sets";
"credentials" includes 1 the wallet has, 1 it does not, "credential_sets" contains one set with the existing credential in wallet.
3. Wallet handles query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet received query
3. The wallet returns the credential set that matched