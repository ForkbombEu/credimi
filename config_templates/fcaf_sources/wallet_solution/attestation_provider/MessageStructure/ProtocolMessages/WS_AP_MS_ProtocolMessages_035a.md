# WS_AP_MS_ProtocolMessages_035a

## Objective

Verify that the Wallet sends a HTTP Request that contains one `OAuth-Client-Attestation` HTTP header when sending a Pushed Authorization Request, when using Pushed Authorization Requests for issuance.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.4
- [RFC9126] section 2.1
- [IETF draft-attestation-based-client-auth] section 6.1, 6.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.

## Preconditions

1. End-user is engaging with a Credential Issuer using a User-agent.
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully obtained Credential Issuer Metadata.
4. The Wallet sends an HTTP POST Request to the selected Authorization Server's PAR endpoint.

## Test Scenario

1. Verify the HTTP Headers of the received HTTP Request.

## Expected results

1. The HTTP Request contains exactly one HTTP header `OAuth-Client-Attestation`.
