# WS_AP_SM_DeviceBinding_011a

## Objective

Verify that the Wallet Provider signs the Key Attestation using an acceptable signature algorithm, when the Wallet uses the `jwt` Proof Type.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.6.1.2
- [HAIP] section 4.5.1
- [OpenID4VCI] section F.1

## Profile applicability

Wallet uses the `jwt` proof type to convey a key attestation in Credential Requests.

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Technology

Credential with Key Binding.

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully authenticated to the (Authorization Server of the) Credential Issuer.
4. Wallet successfully obtained an Access Token from the Token Endpoint of the (Authorization Server of the) Credential Issuer.
5. Wallet send a syntactically correct Credential Request to the Credential Endpoint of the Credential Issuer.
6. Wallet send a syntactically correct `proofs` parameter in the Credential Request.
7. Wallet send a valid signed JWT as value of the  `jwt` property in the `proofs` parameter of the Credential Request.

## Test Scenario

1. Verify the signature algorithm (`alg` in protected header) used for signing the Key Attestation.

## Expected results

1. The signature algorithm in `alg`:
    1. is on the list of acceptable algorithms [ECCG ACM], and
    2. is one of `ES256`, `ES384`, or `ES512`.
