# WS_RP_MS_CredentialFormats_033c

## Objective

Verify that the Wallet includes the `status_list` in the `status` element in a mdoc presentation, when the issuer included the status information in the credential in Token Status List format.

## References

- [CIR 2024/2982 amended] annex II
- [ETSI TS 119 472-1] section 6.2.10
- [Token Status List] section 6.3

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Credential in mdoc format
Credential with revocation status information
Credential with Revocation status information in Token Status List Format

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Verifier requested a credential to be presented, using a valid, trusted request for 'default_credential_A' and requesting presentation in mdoc of all data elements.
3. The presentation value of the credential is a syntactically correct presentation of a mdoc.
4. The presentation value of the credential contains a valid signed MSO.
5. The MSO of the presented mdoc credential contains a correctly formatted value `status` element.

## Test Scenario

1. Verify the `status` element in the MSO of the presented mdoc credential.

## Expected results

1. The `status` element in the MSO of the presented mdoc contains a `status_list` key.

## Comments

- While the contents of the `status` claim is a responsibility of the issuer, the Wallet must pass the value unmodified.
