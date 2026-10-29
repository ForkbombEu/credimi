# Shared_JWT_JWE_01

## Objective

Verify the syntactic structure of a JSON Web Encryption using the JWE Compact Serialization is correct.

## References

- [RFC7516] section 7.1
- [RFC4648] section 5

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JWE structure using JWE Compact Serialization is available.

## Test Scenario

1. Verify the number of single period characters (".") in the JWE.
2. Verify that the five parts of the JWE (as separated by the period characters) are base64url-encoded values as specified in RFC 4648 section 5.

## Expected results

1. The number is equal to 4.
2. This is the case. The five parts do not contain line breaks, whitespace, or other additional characters.
