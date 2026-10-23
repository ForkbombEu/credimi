# WS_AP_MS_ProtocolMessages_053d

## Objective

Verify that the Wallet sends a correctly formatted `nonce` claim in the proof with Key Attestation in the Credential Request, when the `jwt` Proof Type is used.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.6.1.2
- [HAIP] section 4.5.1
- [OpenID4VCI] section F.1
- [RFC7519] section 4

## Profile applicability

Wallet uses the `jwt` proof type to convey a key attestation in Credential Requests.

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Credential with Key Binding.

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully authenticated to the (Authorization Server of the) Credential Issuer.
4. Wallet successfully obtained an Access Token from the Token Endpoint of the (Authorization Server of the) Credential Issuer.
5. Wallet send a syntactically correct Credential Request to the Credential Endpoint of the Credential Issuer.
6. Wallet send a valid signed proof with Key Attestation in the Credential Request.
7. the proof with Key Attestation contains a `nonce` claim.

## Test Scenario

1. Verify the `nonce` claim in the proof with Key Attestation.

## Expected results

1. The value of the `nonce` claim is a string.
