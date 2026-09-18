# WS_RP_IA_Engagement_001a

## Objective

Verify that, in a presentation flow via Redirects, the Wallet can be invoked through the custom URL scheme `haip-vp://`, if supported by the Wallet.

## References

- [HAIP] section 5.1
- [OpenID4VP] section 9

## Profile applicability

Wallet supports to be invoked through the custom URL scheme `haip-vp://`.

## EUDI-wallet relevancy

EUDI_generic | EUDI_optional

## Technology

Presentation via Redirects

## Preconditions

1. The End-user is engaging with a Verifier using a User-agent.

## Test Scenario

1. Verifier provides presentation link to the User-agent using URL scheme `haip-vp://`.
    NOTE: The EUDI Wallet should not support this redirects-based mechanism for cross-device presentation flows, e.g. by presenting the link as QR-code; implementing it for cross-device use should not by itself trigger non-compliance.
2. End-user triggers the link to be followed (e.g. clicks a button or link).

## Expected results

1. User-agent presents an option to the End-user to engage the presentation.
2. Wallet is invoked successfully.
