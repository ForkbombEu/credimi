# Shared_JWT_JWS_03

## Objective

Verify that a JOSE Header is a valid JWS Protected Header.

## References

- [RFC7515] section 4

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JWS structure using JWS Compact Serialization is available.
2. The JWS structure is syntactically correct.
3. The JOSE Header has the correct syntactic structure of a JWS protected header.

## Test Scenario

1. Verify that the JOSE Header is a JSON object as specified in RFC 8259 section 4.
2. Verify the number of members of the JOSE Header.
3. Verify the members of the JOSE Header.

## Expected results

1. The JOSE Header is a JSON object.
2. The JOSE header has at least one member.
3. All members of the JOSE Header have one of the following names (and presence):
    1. alg (mandatory)
    2. (optionally) any registered Header Parameter name according to RFC 7515 section 4.1, or a Public Header Parameter name registered in the IANA "JSON Web Signature and Encryption Header Parameters" registry (for JWS).
    3. (optionally) any Private Header Parameter.
