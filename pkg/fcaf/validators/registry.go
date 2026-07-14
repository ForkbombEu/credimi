// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "fmt"

type Registry struct {
	validators map[string]Validator
}

func NewRegistry(validators ...Validator) (*Registry, error) {
	registry := &Registry{validators: map[string]Validator{}}
	for _, validator := range validators {
		if err := registry.Register(validator); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func DefaultRegistry() (*Registry, error) {
	return NewRegistry(
		EvidencePresentValidator{},
		EvidenceNonEmptyValidator{},
		EvidenceMinimumItemsValidator{},
		JSONFieldRequiredValidator{},
		JSONFieldEqualsValidator{},
		JSONFieldPresenceValidator{},
		JWTHeaderFieldEqualsValidator{},
		JWTPayloadFieldEqualsValidator{},
		JWTPayloadObjectKeysAllowedValidator{},
		JWTPayloadFieldPresenceValidator{},
		MDocNamespaceElementPresentValidator{},
		PIDMDocTypeValidator{},
		PIDMDocMandatoryElementsValidator{},
		MDocDigestAlgorithmValidator{},
		MDocElementCBORTypeValidator{},
		MDocElementUTF8StringValidator{},
		MDocElementDateEncodingValidator{},
		MDocElementDateFormatValidator{},
		MDocElementValidDateValidator{},
		MDocElementCountryCodeValidator{},
		MDocElementRFC5322EmailValidator{},
		MDocElementInternationalPhoneValidator{},
		MDocElementStringArrayValidator{},
		MDocElementCountryCodeArrayValidator{},
		MDocElementMapShapeValidator{},
		MDocElementMapTextValuesValidator{},
		MDocElementMapMemberCountryCodeValidator{},
		MDocElementMapMemberUTF8MaxLengthValidator{},
		MDocElementUnsignedIntegerAllowedValidator{},
		MDocElementJPEGValidator{},
		MDocDomesticNamespaceValidator{},
		MDocElementCountrySubdivisionValidator{},
		JOSEJWEEncryptedResponseValidator{},
		OID4VPDeviceBindingValidator{},
		OID4VPNonceStateBindingValidator{},
		DCQLResponseConstraintsValidator{},
		SDJWTClaimPresentValidator{},
		SDJWTClaimTypeValidator{},
		SDJWTClaimStringPrefixValidator{},
		SDJWTClaimUTF8StringValidator{},
		SDJWTClaimRFC5322EmailValidator{},
		SDJWTClaimNonEmptyUTF8StringValidator{},
		SDJWTClaimInternationalPhoneValidator{},
		SDJWTClaimCountryCodeValidator{},
		SDJWTClaimDateFormatValidator{},
		SDJWTClaimValidDateValidator{},
		SDJWTClaimStringArrayValidator{},
		SDJWTClaimCountryCodeArrayValidator{},
		SDJWTClaimObjectValidator{},
		SDJWTClaimObjectKeysValidator{},
		SDJWTClaimObjectStringValuesValidator{},
		SDJWTClaimNestedStringMaxLengthValidator{},
		SDJWTClaimIntegerAllowedValidator{},
		SDJWTClaimJPEGDataURLValidator{},
		SDJWTClaimCountrySubdivisionValidator{},
		SDJWTDomesticNamespaceValidator{},
		SDJWTIssuerX509HeaderValidator{},
		PIDSDJWTVCTValidator{},
		PIDSDJWTMandatoryClaimsValidator{},
	)
}

func (r *Registry) Register(validator Validator) error {
	if validator == nil {
		return fmt.Errorf("validator is nil")
	}
	id := validator.ID()
	if id == "" {
		return fmt.Errorf("validator id is required")
	}
	if _, exists := r.validators[id]; exists {
		return fmt.Errorf("duplicate validator id %q", id)
	}
	r.validators[id] = validator
	return nil
}

func (r *Registry) Get(id string) (Validator, bool) {
	if r == nil {
		return nil, false
	}
	validator, ok := r.validators[id]
	return validator, ok
}
