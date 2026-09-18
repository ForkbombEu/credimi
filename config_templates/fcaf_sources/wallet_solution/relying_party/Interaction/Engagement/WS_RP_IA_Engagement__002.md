# WS_RP_IA_Engagement_002

## Objective

Verify that the Wallet supports invocation via the W3C Digital Credentials API or an equivalent platform API using OpenID4VP and HAIP.

## References

- [ETSI TS 119 472-2] section 6.5
- [HAIP] section 5.2
- [OpenID4VP] section A

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation via the W3C Digital Credentials API or an equivalent platform API

## Preconditions

1. End-user is engaging with a Verifier using a User-agent.

## Test Scenario

1. Verifier provides a signed OpenID4VP Authorization Request to the User-agent for use via the W3C Digital Credentials API or an equivalent platform API.
2. End-user triggers the request (e.g. clicks a button).

## Expected results

1. User-agent presents an option to the End-user to engage the presentation.
2. Wallet is invoked successfully through the W3C Digital Credentials API or an equivalent platform API.
