# WS_AP_MS_ProtocolMessages_038a

## Objective

Verify that the Wallet sends a syntactically correct Wallet Instance Attestation Proof of Possesion, when using Pushed Authorization Requests for issuance.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section E
- [RFC9126] section 2.1
- [IETF draft-attestation-based-client-auth] section 5.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. End-user is engaging with a Credential Issuer using a User-agent.
3. Wallet started engagement with Credential Issuer.
4. Wallet successfully obtained Credential Issuer Metadata.
5. Wallet send a HTTP POST Request for a Pushed Authorization Request to the PAR Endpoint of the selected Authorization Server.
6. Wallet send a correctly signed and trusted Wallet Instance Attestation as a Client Attestation.

## Test Scenario

1. Verify the claims set in the Wallet Instance Attestation Proof of Possession JWT.

## Expected results

1. The Proof of Possession contains the following top-level claims:
    1. `iss`
    2. `aud`
    3. `jti`
    4. `iat`
    5. (optionally) `challenge`
    6. (optionally) `nbf`
