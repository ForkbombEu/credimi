# WS_RP_MS_ProtocolMessages_087

## Objective

Test that the wallet can handle Multiple Credential Queries in a request which MAY request a presentation of the same Credential.

## References

- [OpenID4VP] Section 6.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

None

## Test Scenario

1. The Wallet engages with the Verifier.
2. The Verifier sends two separate credential queries in one request, where both are matched to the same credential in the wallet.
3. The wallet handles user consent and response.

## Expected results

1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet recognises that one credential satisfies both queries and MUST NOT return an error for this scenario, or force the user to "pick" the credential twice.
