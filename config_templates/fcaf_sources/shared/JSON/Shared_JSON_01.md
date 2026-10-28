# Shared_JSON_01

## Objective

Verify the syntactic structure of a JSON structure is valid.

## References

- [RFC8259] section 9

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A JSON structure is available.

## Test Scenario

1. Parse the complete JSON structure, using a generic JSON decoder.

## Expected results

1. Parsing is successful and the JSON structure is valid according to RFC 8259.

## Comments

Some decoders allow for incremental parsing. This mode should not be used.
