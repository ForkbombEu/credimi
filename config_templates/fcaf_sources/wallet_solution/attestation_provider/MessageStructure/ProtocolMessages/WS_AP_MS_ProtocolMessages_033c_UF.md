# WS_AP_MS_ProtocolMessages_033c_UF

## Objective

Verify that the Wallet does not engage issuance based on a Credential Offer with the `credential_configuration_ids` property omitted, using the issuer initiated flow and credential offer by value.

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4.2
- [OpenID4VCI] section 4.1.1

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
    1. has a `credential_issuer` property with a valid issuer identifier.
    2. has no `credential_configuration_ids` property.
2. End-user triggers link to be followed.

## Expected results

1. User-agent presents option to End-user to engage issuance.
2. The Wallet does not engage for issuance. This includes:
    1. The Wallet could inform the End-user about the incorrect engagement, if applicable.
