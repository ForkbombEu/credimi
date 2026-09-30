# WS_RP_DM_Credentialmetadata_Domestic_PID_IETF-sd-jwt-vc_001

## Objective

This test case verifies domestic elements in an appropriate namespace are present, if these have been indicated in the ICS.

## References

- [PID rulebook] Annex 3.01, Section 4.2

## Profile applicability

The EUDI wallet contains a Credential in IETF SD-JWT VC format. `vct` claim includes base type of person identification "urn:eudi:pid:1", which contains data elements in a domestic namespace.

## EUDI-wallet relevancy

EUDI_specific | EUDI_required

## Preconditions

1. A presentation request was sent to the EUDI wallet, to retrieve a PID Credential in IETF SD-JWT VC format.
2. All mandatory data elements within namespace "urn:eudi:pid:" and all data elements indicated as present in the ICS were requested.
3. EUDI wallet presented the Credential successfully.

## Test Scenario

1. Verify claims in the domestic namespace are present.

## Expected results

1. One or more data claims in the indicated namespace are present.
