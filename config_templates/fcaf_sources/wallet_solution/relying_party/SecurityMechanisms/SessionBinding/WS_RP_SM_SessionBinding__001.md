# WS_RP_SM_SessionBinding_001

## Objective

Verify that when the Wallet receives an Authorization Request that contains a valid nonce parameter, the wallet will accept the Authorization Request.

## References

- [OpenID4VP] Sections 5.2, 14.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

None

## Test Scenario

1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request containing a valid nonce parameter.

## Expected results

1. Wallet-verifier interaction is successfully initiated.
2. The Wallet successfully receives the Authorization Request.
