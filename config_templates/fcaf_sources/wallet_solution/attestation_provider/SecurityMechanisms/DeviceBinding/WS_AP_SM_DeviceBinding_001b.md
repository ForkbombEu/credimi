# WS_AP_SM_DeviceBinding_001b

## Objective

Verify that the Wallet correctly serializes the Client Attestation Proof of Possession as a signed JWT to authenticate at the PAR Endpoint of the Authorization Server, when using Pushed Authentication Requests in the Authorization Code Flow for issuance.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.4, E
- [IETF draft-attestation-based-client-auth] section 5.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Credential Issuance using the Authorization Code Flow.

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully obtained Credential Issuer Metadata.
4. Wallet send an HTTP POST Request for a Pushed Authorization Request to the PAR Endpoint of the selected Authorization Server.
5. The HTTP Request to the PAR Endpoint of the Authorization Server contains a syntactically correct `OAuth-Client-Attestation-PoP` HTTP Header.

## Test Scenario

1. For the value of the `OAuth-Client-Attestation-PoP` HTTP Header in the HTTP request, perform all Shared_JWT_JWE test cases.

## Expected results

1. All test cases pass.
