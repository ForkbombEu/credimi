# Shared_JSON_03

## Objective

Verify that the JSON structure is encoded using UTF-8 and does not contain a byte order mark.

## References

- [RFC8259] section 8.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. A syntactically correct JSON structure is available.

## Test Scenario

1. Verify that the JSON structure is encoded using UTF-8.
2. Verify that the JSON structure does not contain a byte order mark.

## Expected results

1. The JSON structure is encoded using byte sequence valid for UTF-8 and only encodes code points specified in Unicode.
2. The JSON structure does not contain a byte order mark (U+FEFF) at the beginning.
