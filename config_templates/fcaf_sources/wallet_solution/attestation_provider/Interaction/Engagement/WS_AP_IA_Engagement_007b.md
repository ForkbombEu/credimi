# WS_AP_IA_Engagement_007b

## Objective

Verify that, for issuance via Redirects, the Wallet can be invoked through the `eu-eaa-offer://` custom URL scheme.

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Technology

Wallet supports the Issuer initiated issuance flow.
Issuance via Redirects.
Wallet invocation uses the required custom URL scheme `eu-eaa-offer://`.

## Preconditions

1. End-user is engaging with a Issuer using a User-agent.

## Test Scenario

1. Credential Issuer provides Credential Offer link to the User-agent using URL scheme `eu-eaa-offer://`.
    Note: for engagement by presenting the link as QR-code for cross-device usage, additional security measures are recommended.
2. End-user triggers link to be followed (e.g. clicks button or link).

## Expected results

1. User-agent presents option to End-user to engage issuance.
2. Wallet is invoked successfully.
