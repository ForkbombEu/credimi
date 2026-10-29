# WS_AP_MS_ProtocolMessages_052a

## Objective

Verify that the Wallet sends a syntactically correct header of the proof with Key Attestation in the Credential Request, when the `jwt` Proof Type is used.

## References

- [CIR 2024/2979 amended] annex Ib
- [ETSI TS 119 472-3] section 4.6.1.2
- [HAIP] section 4.5.1
- [OpenID4VCI] section F.1
- [RFC7515] section 4

## Profile applicability

Wallet uses the `jwt` proof type to convey a key attestation in Credential Requests.

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully authenticated to the (Authorization Server of the) Credential Issuer.
4. Wallet successfully obtained an Access Token from the Token Endpoint of the (Authorization Server of the) Credential Issuer.
5. Wallet send a syntactically correct Credential Request to the Credential Endpoint of the Credential Issuer.
6. Wallet send a valid signed proof with Key Attestation in the Credential Request.

## Test Scenario

1. Verify the presence of parameters in the header of the proof with Key Attestation.

## Expected results

1. The presence of parameters in the header of the proof with Key Attestation is as follows:
    1. `alg` is present.
    2. `typ` is present.
    3. (optionally) `kid` can be present.
    4. (optionally) `jwk` can be present.
    5. `key_attestation` is present.
    6. (optionally) `x5c` can be present.
    7. (optionally) `trust_chain` can be present.
