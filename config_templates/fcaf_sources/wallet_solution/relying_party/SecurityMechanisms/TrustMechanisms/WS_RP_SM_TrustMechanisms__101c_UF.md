# WS_RP_SM_TrustMechanisms__101c_UF

## Objective

Verify that the Wallet does not present a credential to a Relying Party, if a Wallet Relying Party Registration Certificate is used that does not match to the same organization of the WRPAC used, when using OpenID4VP presentation.

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
            1. the WRPRC's `sub` field has a different value from the WRPAC's `organizationIdentifier`.
            2. the WRPRC does not contain a `usesIntermediary` field.

## Expected results

1. Wallet does not complete the presentation interaction, and
    1. informs, if applicable, the user of not having the requested credential, and
    2. aborting the interaction with the Verifier, either by
        1. responding with an error `invalid_request`, or
        2. responding by returning an error without any details, or
        3. discontinuing the interaction, e.g. by closing the communication channel, if applicable.
    3. if any response is shared to the Verifier, the response does not include a `vp_token`.
