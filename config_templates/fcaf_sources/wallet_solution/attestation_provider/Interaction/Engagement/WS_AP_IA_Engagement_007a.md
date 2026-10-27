# WS_AP_IA_Engagement_007a

## Objective

Verify that, for issuance via Redirects, the Wallet can be invoked through the `haip-vci://` custom URL scheme, if supported by Wallet.

## References

- [ETSI TS 119 472-3]  section 4.1
- [HAIP] section 4.2

## Profile applicability

Wallet supports invocation through the `haip-vci://` custom URL scheme.

## EUDI-wallet relevancy

EUDI_generic | EUDI_optional

## Technology

Issuance via Redirects.
Wallet supports the Issuer initiated issuance flow.

## Preconditions

1. End-user is engaging with a Issuer using a User-agent.

## Test Scenario

1. Credential Issuer provides Credential Offer link to the User-agent using URL scheme `haip-vci://`.
    Note: for engagement by presenting the link as QR-code for cross-device usage, additional security measures are recommended.
2. End-user triggers link to be followed (e.g. clicks button or link).

## Expected results

1. User-agent presents option to End-user to engage issuance.
2. Wallet is invoked successfully.
