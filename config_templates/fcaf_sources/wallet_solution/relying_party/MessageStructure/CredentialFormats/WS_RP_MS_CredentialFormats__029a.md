# WS_RP_MS_CredentialFormats_029a

## Objective

Verify that the Wallet includes the `status` claim in a SD-JWT VC presentation, when the issuer included the status information in the credential.

## References

- [ETSI TS 119 472-1] section 5.2.10
- [Token Status List] section 6.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Credential in SD-JWT VC format
Credential with revocation status information

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Verifier requested a credential to be presented, using a valid, trusted request for 'default_credential_A' and requesting presentation in SD-JWT VC format of all claims.
3. The presentation value of the credential is a syntactically correct serialization in compact serialization format of an SD-JWT VC.
4. The presentation value of the credential contains a valid signed SD-JWT.

## Test Scenario

1. Verify the presented SD-JWT VC credential.

## Expected results

1. The presented SD-JWT VC contains a top-level `status` claim in the body of the SD-JWT.
