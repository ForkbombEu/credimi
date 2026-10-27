# Shared_JWT_JWS_05

## Objective

Verify that a JWT Payload is a valid JSON Web Token.

## References

- [RFC7519] section 4

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JWS structure using JWS Compact Serialization is available.
2. The JWS structure is syntactically correct.
3. The JWS Payload has the correct syntactic structure of a JSON Web Token.

## Test Scenario

1. Verify the members of the JWS Payload.

## Expected results

1. All members of the JWS Payload (i.e. the JWT Claims Set) have one of the following names:
    1. A Registered Claim name according to RFC 7519 section 4.1, i.e. `iss`, `sub`, `aud`, `exp`, `nbf`, `iat` or `jti`.
    2. A Public Claim name registered in the IANA "JSON Web Token Claims" registry.
    3. One of the claims specified for EUDI-wallet, e.g. in a relevant ETSI specification, such as `issuing_authority` or `category`.
    4. (optionally) any other Private Claim.
