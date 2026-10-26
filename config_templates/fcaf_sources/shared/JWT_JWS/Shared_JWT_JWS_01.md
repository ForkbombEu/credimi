# Shared_JWT_JWS_01

## Objective

Verify the syntactic structure of a JSON Web Signature using the JWS Compact Serialization is correct.

## References

- [RFC7515] section 7.1
- [RFC4648] section 5

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JWS structure using JWS Compact Serialization is available.

## Test Scenario

1. Verify the number of single period characters (".") in the JWS.
2. Verify that the three parts of the JWS (as separated by the period characters) are base64url-encoded values as specified in RFC 4648 section 5.

## Expected results

1. The number is equal to 2.
2. This is the case. The three parts do not contain line breaks, whitespace, or other additional characters.
