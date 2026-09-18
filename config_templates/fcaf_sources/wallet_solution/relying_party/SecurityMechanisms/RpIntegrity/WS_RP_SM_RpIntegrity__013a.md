# WS_RP_SM_RpIntegrity_013a

## Objective

Verify that the Wallet accepts a signed Request Object, when using OpenID4VP presentation via redirects and Request Object by reference, if the request signature is valid.

## References

- [ETSI TS 119 472-2] section 6.4.2
- [HAIP] section 5.1
- [OpenID4VP] section 5
- [RFC9101]

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation via Redirects

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet trusts the CA issuing a Relying Party's WRPAC (i.e. through the EUDI trust lists).
3. Wallet and Verifier are engaged, and a presentation using redirects and Request Object by reference has been triggered.

## Test Scenario

1. Verifier sends a Request Object to the Wallet upon Wallet requesting the Request Object. The Request Object:
    1. Contains all required parameters with valid values to request 'default_credential_A'.
    2. Is signed using the private key corresponding the Verifier's WRPAC.
    3. Is signed using an acceptable ([ECCG ACM] approved) algorithm.
    4. Is in JAR format, which is also a JWT.
    5. Has a valid signature.

## Expected results

1. Wallet responds with a presentation of 'default_credential_A'.
