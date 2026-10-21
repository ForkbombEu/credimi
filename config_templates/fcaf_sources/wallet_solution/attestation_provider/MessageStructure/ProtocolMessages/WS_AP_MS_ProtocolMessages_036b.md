# WS_AP_MS_ProtocolMessages_036b

## Objective

Verify that the Wallet sends a HTTP Request that contains a syntactically correct `OAuth-Client-Attestation-PoP` HTTP header when sending a Pushed Authorization Request, when using Pushed Authorization Requests for issuance.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.4
- [RFC9126] section 2.1
- [IETF draft-attestation-based-client-auth] section 5.2, 6.1

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
5. The HTTP Request to the selected Authorization Server's PAR endpoint contains an `OAuth-Client-Attestation-PoP` HTTP header.

## Test Scenario

1. Verify the `OAuth-Client-Attestation-PoP` HTTP-Header of the received HTTP Request.

## Expected results

1. The value of the `OAuth-Client-Attestation-PoP` HTTP Header of the HTTP Request has the structure of a signed JWT in compact serialization, that is three Base64url-encoded parts separated by exactly two period (`.`) characters.
