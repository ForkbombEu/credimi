# Shared_JWT_JWE_02

## Objective

Verify the syntactic structure of the protected header of a JSON Web Encryption using the JWE Compact Serialization is correct.

## References

- [RFC7516] section 5.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JWE structure using JWE Compact Serialization is available.
2. The syntactic structure of that JWE structure is correct.

## Test Scenario

1. Take the first part of the JWE and Base64url-decode it to obtain the JWE Protected Header.
2. For the JWE Protected Header, perform all Shared_JSON test cases.

## Expected results

1. Decoding is successful.
2. All test cases pass.
