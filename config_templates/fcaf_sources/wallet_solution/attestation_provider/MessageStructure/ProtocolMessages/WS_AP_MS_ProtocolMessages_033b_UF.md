# WS_AP_MS_ProtocolMessages_033b_UF

## Objective

Verify that the Wallet does not engage issuance based on a Credential Offer with the `credential_issuer` omitted, using the issuer initiated flow and credential offer by value.

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4.2
- [OpenID4VCI] sections 4.1.1 and 12.2.2

## Profile applicability

Wallet supports the Issuer initiated issuance flow.

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Wallet supports invocation through the custom URL scheme `eu-eaa-offer://`.

## Preconditions

1. End-user is engaging with a Credential Issuer using a User-agent.
2. Wallet can be invoked for issuance using a custom URL scheme.

## Test Scenario

1. Credential Issuer provides a Credential Offer as parameter in a custom URL scheme to the User-agent. The `credential_offer` parameter:
    1. has no `credential_issuer` property.
    2. has a `credential_configuration_ids` property, as an array containing at least one identifier as string.
2. End-user triggers link to be followed.

## Expected results

1. User-agent presents option to End-user to engage issuance.
2. The Wallet does not engage for issuance. This includes:
    1. The Wallet could inform the End-user about the incorrect engagement, if applicable.
    2. The Wallet does not connect to the Credential Issuer, for retrieving metadata. This includes not connecting to the Endpoint created by deriving from the URL where the User-agent was engaged (e.g. referrer or origin), by inserting `.well-known/openid-credential-issuer`.
