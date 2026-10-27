# Shared_JWT_JWS_04

## Objective

Verify that the syntactic structure of the JWT Payload of a JSON Web Token with JSON Web Signature using the JWS Compact Serialization is correct.

## References

- [RFC7519] section 4
- [RFC7515] section 7.1
- [RFC4648] section 5

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JWS structure using JWS Compact Serialization is available.
2. The JWS structure is syntactically correct.

## Test Scenario

1. Take the second part of the JWS and Base64url-decode it to obtain the JWS Payload.
2. For the JWS Payload, perform all Shared_JSON test cases.

## Expected results

1. Decoding is successful.
2. All test cases pass.
