# WS_RP_SM_SessionEncryption_013

## Objective

Verify that the Wallet includes only valid disclosures when presenting an SD-JWT VC, using OpenID4VP.

## References

- [HAIP] section 6, 8
- [OpenID4VP] section 8
- [SD-JWT VC] section 4
- [SD-JWT] section 4.1.1, 4.2.3, 7.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation using OpenID4VP
Credential in SD-JWT VC format
Issuer uses compact serialization of SD-JWT and SD-JWT VC

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Verifier requested a credential to be presented, using a valid, trusted request for 'default_credential_A' and requesting presentation in SD-JWT VC format of all claims.
3. The Wallet transmitted a syntactically correct presentation in the `vp_token` of the Authorization Response.
4. The presentation value of the credential is a syntactically correct serialization in compact serialization format of an SD-JWT VC.
5. The presentation value of the credential contains a correctly serialized SD-JWT.
6. The presentation value of the credential contains correctly serialized disclosures.
7. The SD-JWT contains a `_sd_alg` valid and acceptable.

## Test Scenario

1. Select the disclosure digest algorithm from `_sd_alg` in the SD-JWT of the presentation.
2. Find all selective disclosures in the SD-JWT (see [RFC9901] section 7.1, step 3 b).
       NOTE: apply recursion on substep 2 and 3 above, as described in [RFC9901] section 7.1, step 3 b and c, when applicable.
3. Verify each Disclosure value. That is:
    1. Calculate the digest over the Disclosure. That is apply the disclosure digest algorithm to the base64url-encoded value of the disclosure.
    2. Check the digest value of a selective disclosure in the SD-JWT matches one of the selective disclosures found in the SD-JWT.
    3. Process the Disclosure as in [RFC9901] section 7.1, step 3. c. step ii and iii). If the Disclosure needs to be rejected, fail this test case.
       NOTE: apply recursion on substep 2 and 3 above, as described in [RFC9901] section 7.1, step 3 b and c, when applicable.

## Expected results

1. The selective disclosure algorithm is available for processing next test steps.
2. Selective disclosures in the SD-JWT exist.
3. All Disclosures have a matching selective disclosure in the SD-JWT.
