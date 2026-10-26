# Shared_JWT_JWE_03

## Objective

Verify that a JOSE Header is a valid JWE Protected Header.

## References

- [RFC7516] section 4.1
- [RFC8259] section 4

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JWE structure using JWE Compact Serialization is available.
2. The syntactic structure of that JWE structure is correct.
3. The syntactic structure of the protected header of that JWE structure is correct.

## Test Scenario

1. Verify that the JOSE Header is a JSON object as specified in RFC 8259 section 4.
2. Verify the number of members of the JOSE Header.
3. Verify the members of the JOSE Header.

## Expected results

1. This is the case.
2. The JOSE header has at least two members.
3. All members of the JOSE Header have one of the following names:
   1. Mandatorily: `alg`,
   2. Mandatorily: `enc`,
   3. Optionally: a Registered Header Parameter name according to RFC 7516 section 4.1, or a Public Header Parameter name registered in the IANA "JSON Web Signature and Encryption Header Parameters" registry (for JWE).
   4. Optionally: any Private Header Parameter.
