# WS_AP_SM_DeviceBinding_003a

## Objective

Verify that the Wallet uses an acceptable signature algorithm for signing the Client Attestation Proof of Possession used by the Wallet to authenticate at the PAR Endpoint of the Authorization Server, when using Pushed Authentication Requests in the Authorization Code Flow for issuance.

## References

- [CIR 2024/2979 amended] annex Ia, Ib
- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.4, E
- [IETF draft-attestation-based-client-auth] section 5.2
- [ECCG ACM] section 5.2

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
5. The HTTP Request to the PAR Endpoint of the Authorization Server contains a syntactically correct `OAuth-Client-Attestation-PoP` HTTP header.
6. The Client Attestation Proof of Possession used to authenticate to the PAR Endpoint is a correctly serialized signed JWT.

## Test Scenario

1. Verify the signature algorithm (`alg` in protected header) used for signing the Client Attestation Proof of Possession.

## Expected results

1. The signature algorithm in `alg`:
    1. is on the list of acceptable algorithms [ECCG ACM], and
    2. is one of `ES256`, `ES384`, or `ES512`.
