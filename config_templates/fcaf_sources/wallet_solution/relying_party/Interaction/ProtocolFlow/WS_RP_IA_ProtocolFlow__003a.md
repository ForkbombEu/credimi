# WS_RP_IA_ProtocolFlow_003a

## Objective

Verify that the Wallet, in the presentation flow via DC API using OpenID4VP, supports the Response mode `dc_api.jwt`.

## References

- [HAIP] section 5.2
- [OpenID4VP] section 8.3, A.2
- [DC API]

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation via the W3C Digital Credentials API or equivalent platform API.
Presentation using OpenID4VP

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. End-user interacts with Verifier using a User-agent.

## Test Scenario

1. The Verifier provides a OpenID4VP Request Object to the User-agent for usage with the W3C DC API or equivalent platform API, where the Request Object:
    1. is requesting presentation of `default_credential_A`,
    2. has the `response_mode` parameter set to the value `dc_api.jwt`,
    3. having all other required parameters with valid values,
    4. and valid signed.
2. End-user triggers the mediating API to initiate the provided Request Object.
3. Verify the Authorization Response received.

## Expected results

1. User-agent presents option to End-user to engage presentation.
2. The Wallet is invoked.
3. The Authorization Response is made available as response through the W3C DC API or equivalent platform API as an encrypted JWT, that is a series of 5 sets of base64url-encoded concatenated by periods ('.').

## Comments

- Note: validation of the encrypted JWT itself as Authorization Response, is done in other test cases.
