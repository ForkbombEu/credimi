# WS_AP_MS_ProtocolMessages_037a

## Objective

Verify that the Wallet sends a syntactically correct body of the Wallet Instance Attestation as Client Attestation, when using Pushed Authorization Requests for issuance.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section E
- [IETF draft-attestation-based-client-auth] section 5.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

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

1. Verify the contents of the Wallet Instance Attestation (i.e. the JWT body).

## Expected results

1. The WIA contains the following top-level claims:
    1. `wallet_name`
    2. `wallet_version`
    3. `wallet_solution_certification_information`
    4. `client_status`
    5. `exp`
    6. `wallet_link`
    7. `iss`
    8. `sub`
    9. `cnf`
    10. (optionally) `iat`
    11. (optionally) `nbf`
