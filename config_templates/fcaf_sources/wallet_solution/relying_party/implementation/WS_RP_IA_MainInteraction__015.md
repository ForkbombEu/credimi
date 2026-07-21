# WS_RP_IA_MainInteraction_015

## Objective
A Credential presentation may include "extra" claims not selected according to rules, if they are non-selective (fixed) technical fields that cannot be hidden.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet has an SD-JWT credential. By definition, the vct (type) and iss (issuer) fields are fixed/mandatory and cannot be hidden.

## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query asking only for given_name.
3. Response

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet selects given_name but must also include the mandatory envelope data required to make the credential cryptographically valid.
3. The vp_token contains given_name PLUS the mandatory technical fields (e.g., vct, iss, iat).