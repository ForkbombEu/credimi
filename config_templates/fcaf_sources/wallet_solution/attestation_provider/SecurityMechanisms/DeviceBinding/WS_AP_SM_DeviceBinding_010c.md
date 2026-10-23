# WS_AP_SM_DeviceBinding_010c

## Objective

Verify that the Wallet uses a valid signature value for signing the proof with Key Attestation in the Credential Request to the Credential Endpoint of a Credential Issuer, when the `jwt` Proof Type is used.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.6.1.2
- [HAIP] section 4.5.1
- [OpenID4VCI] section 8.2, F.1
- [ECCG ACM] section 5.2

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
6. Wallet send a syntactically correct `proofs` parameter in the Credential Request.
7. Wallet send a JWT as value of the `jwt` property in the `proofs` parameter of the Credential Request.
8. The proof with Key Attestation uses an acceptable signature algorithm.

## Test Scenario

1. Verify the signature value of the proof with Key Attestation, using the public key in the first element in `attested_keys` in `key_attestation`.

## Expected results

1. The signature is valid.
