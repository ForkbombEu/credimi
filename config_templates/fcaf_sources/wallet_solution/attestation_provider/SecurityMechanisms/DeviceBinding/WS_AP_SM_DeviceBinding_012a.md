# WS_AP_SM_DeviceBinding_012a

## Objective

Verify that the Wallet uses the nonce from the Nonce Endpoint of the Credential Issuer to include a nonce in the proof of Key Attestation, when the Wallet uses the `jwt` Proof Type.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.6.1.2
- [HAIP] section 4.5.1
- [OpenID4VCI] section 7, 8.2, F.1

## Profile applicability

Wallet uses the `jwt` proof type to convey a key attestation in Credential Requests.

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Technology

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully obtained Credential Issuer Metadata, including the `nonce_endpoint` of the Credential Issuer.
4. Wallet successfully authenticated to the (Authorization Server of the) Credential Issuer.
5. Wallet successfully obtained an Access Token from the Token Endpoint of the (Authorization Server of the) Credential Issuer.
6. Wallet successfully obtained a value in `c_nonce` in the Nonce Response, by making a Nonce Request to the Nonce Endpoint of the Credential Issuer.
7. Wallet send a valid and trusted signed JWT as proof of Key Attestation, with the Credential Request.
8. Wallet send a syntactically correct `nonce` in the proof of Key Attestation.

## Test Scenario

1. Verify the `nonce` value in the proof of Key Attestation.

## Expected results

1. The value of the `nonce` in the signed JWT that makes the proof of Key Attestation, is equal to the value of the `c_nonce` provided in the Nonce Response from the Nonce Endpoint.
