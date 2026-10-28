# WS_AP_IA_MainInteraction_031b_UF

## Objective

Verify that the Wallet treats a Credential Error Response containing `credential_request_denied` as unrecoverable.

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4
- [OpenID4VCI] section 8.3.1.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

OpenID4VCI Credential Endpoint error-response handling.

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully sent a valid Credential Request to the Credential Issuer.

## Test Scenario

1. The Issuer returns a valid Credential Error Response containing the error code `credential_request_denied`.

## Expected results

1. The Wallet does not store any new Credential. The End-user is informed the Credential was not issued, if applicable. The Wallet aborts the issuance process and does not make any further requests as part of this issuance attempt.
