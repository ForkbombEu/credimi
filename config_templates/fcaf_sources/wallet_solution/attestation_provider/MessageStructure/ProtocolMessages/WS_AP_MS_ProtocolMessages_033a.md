# WS_AP_MS_ProtocolMessages_033a

## Objective

Verify that the Wallet can handle a correctly formatted Credential Offer, using the issuer initiated flow and credential offer by value.

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4.2
- [OpenID4VCI] section 4.1

## Profile applicability

Wallet supports the Issuer initiated issuance flow.

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.

## Preconditions

1. End-user is engaging with a Credential Issuer using a User-agent.
2. Wallet can be invoked for issuance using the custom URL scheme `eu-eaa-offer://`.

## Test Scenario

1. Credential Issuer provides a valid Credential Offer as parameter in a custom URL Scheme to the User-agent. The `credential_offer` parameter:
    1. has a `credential_issuer` property with a valid issuer identifier.
    2. has a `credential_configuration_ids` property, as an array containing at least one identifier as string.
2. End-user triggers link to be followed.

## Expected results

1. User-agent presents option to End-user to engage issuance, optionally after requesting the Credential Issuer Metadata (derived using `.well-known` from `credential_issuer`, if not cached already).
2. The Wallet connects to the Credential Issuer, to make a request at the Authorization Server's Authorization Endpoint.
