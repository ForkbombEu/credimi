# WS_RP_SM_IssuerIntegrity_012

## Objective

Verify that Wallet uses MSO revocation mechanism following requirements defined in ISO/IEC 18013-5 ([ISO.18013-5.second.edition]).

## References

- [HAIP] Section 5.3.1
- TODO: [ISO/IEC 18013-5] MSO revocation

## Profile applicability

Wallet supports ISO mdoc

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. The mdoc's status is set to revoked.

## Test Scenario

1. Wallet and verifier engage.
2. Verifier sends request for a revoked MSO.
3. Wallet sends response.

## Expected results

1. Wallet and verifier successfully engage.
2. Wallet processes request.
3. Wallet returns error to verifier, without revealing reason (revocation of MSO).
