# WS_RP_SM_TrustMechanisms__101

## Objective

Verify that the Wallet accepts a valid Wallet Relying Party Registration Certificate, when using OpenID4VP presentation.

## References

- [ETSI TS 119 472-2] section 6.3.2.2
- [ETSI TS 119 475] section 5.1.3, 5.2.4
- [OpenID4VP] sections 5.1 and 5.11.1

## Profile applicability

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Technology

WRPRC

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet trusts the CA issuing a Relying Party's WRPAC (i.e. through the EUDI trust lists).
3. Wallet trusts the CA issuing a Relying Party's WRPRC (i.e. through the EUDI trust lists).
4. Wallet and Verifier are engaged, and a presentation using OpenID4VP has been triggered.

## Test Scenario

1. Verifier sends a Request Object to the Wallet. The Request Object:
    1. contains all required parameters with valid values to request 'default_credential_A'.
    2. has a valid WRPAC.
    3. is signed by the Verifier using the private key corresponding to the public key in the valid WRPAC.
    4. contains a `verifier_info` property, with the following properties:
        1. has a `format` property with the value `registration_cert`,
        2. has a `data` property with as value a string with the base64url encoding of a valid WRPRC, where
            1. the WRPRC's `sub` field has the same value as the WRPAC's `organizationIdentifier`.
            2. the WRPRC does not contain a `usesIntermediary` field.

## Expected results

1. Wallet responds with a presentation of 'default_credential_A'.
