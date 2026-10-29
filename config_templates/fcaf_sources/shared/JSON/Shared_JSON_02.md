# Shared_JSON_02

## Objective

Verify that any object in the JSON structure does not contain multiple name/value pairs with the same name.

## References

- [RFC8259] section 4

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A syntactically correct JSON structure is available.

## Test Scenario

1. For all JSON objects within the JSON structure, verify whether the object contains multiple name/value pairs with the same name.
## Expected results

1. No object with multiple members with the same name are present in the JSON structure.
