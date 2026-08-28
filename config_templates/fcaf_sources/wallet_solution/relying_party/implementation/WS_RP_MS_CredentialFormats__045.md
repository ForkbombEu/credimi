# WS_RP_MS_CredentialFormats_045

## Objective
Verify that Wallet can handle presenting Credential in IETF SD-JWT VC format using compact serialization as defined in [RFC9901] when there is key binding.

## References
[OIDF.HAIP] section 6.1
[RFC9901]

## Profile applicability
Wallet supports IETF SD-JWT VC with key-binding

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
A) Credential Issuer issued Credential to the Wallet in IETF SD-JWT VC format, using compact serialization as defined in [RFC9901].
B) There is a Key Binding JWT.

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential in IETF SD-JWT VC format.
3. Verify if Wallet presents successfully Credential in IETF SD-JWT VC format using compact serialization. That is, the compact serialized format for the SD-JWT is the concatenation of each part delineated with a single tilde ('~') character as follows, where "D.1" to "D.N" represent the respective Disclosures:
<Issuer-signed JWT>~<D.1>~<D.2>~...~<D.N>~
The order of the concatenated parts is the Issuer-signed JWT, a tilde character, zero or more Disclosures each followed by a tilde character, and lastly the Key Binding JWT.

## Expected results
1. This is the case. 
2. This is the case.
3. Wallet presents successfully Credential in IETF SD-JWT VC format using compact serialization. That is, the compact serialized format for the SD-JWT is the concatenation of each part delineated with a single tilde ('~') character as follows, where "D.1" to "D.N" represent the respective Disclosures:
<Issuer-signed JWT>~<D.1>~<D.2>~...~<D.N>~
The order of the concatenated parts is the Issuer-signed JWT, a tilde character, zero or more Disclosures each followed by a tilde character, and lastly the Key Binding JWT.

