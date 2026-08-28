# WS_RP_MS_ProtocolMessages_024

## Objective
Test that the Wallet returns an error if a request contains even one unrecognized transaction data type or transaction data not conforming to the respective type definition.

## References
[OIDF.OID4VP] section 5.1; [OIDF.OID4VP] section 8.4

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request with a transaction_data array where at least one entry has an unrecognized transaction data type.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet returns an invalid_transaction_data error due to the unrecognized transaction data type; presentation flow is not initiated.

