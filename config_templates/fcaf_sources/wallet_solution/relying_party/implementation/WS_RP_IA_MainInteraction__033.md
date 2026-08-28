# WS_RP_IA_MainInteraction_033

## Objective
Verify that the Wallet correctly evaluates a combination of value constraints in a single DCQL Query, withholding any credentials that fail even a single individual requested criterion.

## References
[OID4VP Section 6.4]

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Profile applicability


## Preconditions
The wallet contains multiple credentials of the same type, provisioned as follows:
Credential A (Full Match): Perfectly satisfies all criteria, including an `expiry` = 2025
Credential B1 (Fails Age): Matches everything else, but `age` = 17.
Credential B2 (Fails Data Type): Matches everything else, but `kg` = 10.5 (float mismatch).
Credential B3 (Fails Case): Matches everything else, but `country` = "Spain" (capitalized).
Credential B4 (Fails Whitespace): Matches everything else, but `name` = "Smith" (missing trailing space).
Credential B5 (Fails Encoding): Matches everything else, but `city` = "Munchen" (missing umlaut).
Credential B6 (Fails Array): Matches everything else, but `array` = ["FR", "DE"] (multi-element array).
Credential B7 (Fails Expiry Boundary): Matches everything else, but `expiry` = 2026 (violates `<=` condition).

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a single DCQL query targeting a single credential with the following combined value constraints:
   * Logical boundaries: `age >= 18` and `expiry <= 2025`
   * Data type strictness: `kg == 10`
   * Case sensitivity: `country == "spain"`
   * Whitespace sensitivity: `name == "Smith "`
   * Character encoding: `city == "München"`
   * Array constraints: Requesting a single-element array `["FR"]`
3. The Wallet processes the request.

## Expected results
1. Wallet and Verifier successfully initiate interaction.
2. The Wallet only prompts the user to release Credential A. 
3. The Wallet completely excludes all target "trap" credentials (B1 through B7) from the user selection prompt.