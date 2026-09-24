# WS_RP_IA_MainInteraction_022

## Objective

Test that Claim_sets MUST NOT be supported if claims is absent.

## References

- [OpenID4VP] Section 6.4

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

None

## Test Scenario

1. Verifier sends a DCQL query with a credential with claim_sets, but missing claims

## Expected results

1. Wallet identifies invalid query, returns error.
