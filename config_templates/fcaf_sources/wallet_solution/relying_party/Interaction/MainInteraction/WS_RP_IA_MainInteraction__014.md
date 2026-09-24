# WS_RP_IA_MainInteraction_014

## Objective

A Credential presentation may include "extra" claims not selected according to rules, if they are requested by a different query within the same session.

## References

- [OpenID4VP] Section 6.4

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet has one ID with Name, Address, and Age.

## Test Scenario

1. The Wallet engages with the Verifier.
2. Verifier sends one request with two DCQL queries: Query 1: Asks for Name. Query 2: Asks for Address
3. Wallet handles query

## Expected results

1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet identifies that both queries can be satisfied by the same credential. It selects the "union" of both requests (Name + Address).
    - The vp_token contains both Name and Address. Age remains hidden.
