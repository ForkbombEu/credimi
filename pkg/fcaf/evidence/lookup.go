// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"reflect"
	"strconv"
	"strings"
)

func Lookup(bundle Bundle, path string) LookupResult {
	root, parts, ok := rootValue(bundle, path)
	if !ok {
		return LookupResult{Path: path, Found: false}
	}
	value, found := descend(root, parts)
	if !found {
		return LookupResult{Path: path, Found: false}
	}
	return LookupResult{
		Path:  path,
		Found: true,
		Value: value,
		Type:  typeName(value),
	}
}

func rootValue(bundle Bundle, path string) (any, []string, bool) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return nil, nil, false
	}
	switch parts[0] {
	case "evidence":
		root, ok := evidenceSlot(bundle, parts[1])
		return root, parts[2:], ok
	case "preconditions":
		return bundle.Preconditions, parts[1:], true
	case "runtime":
		return bundle.Runtime, parts[1:], true
	default:
		return nil, nil, false
	}
}

func evidenceSlot(bundle Bundle, name string) (any, bool) {
	switch name {
	case "raw_request_object":
		return bundle.RawRequestObject, true
	case "decoded_request_object":
		return bundle.DecodedRequestObject, true
	case "raw_presentation_response":
		return bundle.RawPresentationResponse, true
	case "decoded_presentation_response":
		return bundle.DecodedPresentationResponse, true
	case "vp_token":
		return bundle.VPToken, true
	case "presentation_submission":
		return bundle.PresentationSubmission, true
	case "decoded_sdjwt":
		return bundle.DecodedSDJWT, true
	case "mdoc":
		return bundle.MDoc, true
	case "issuer_metadata":
		return bundle.IssuerMetadata, true
	case "verifier_metadata":
		return bundle.VerifierMetadata, true
	case "authorization_server_metadata":
		return bundle.AuthorizationServerMetadata, true
	case "jwks":
		return bundle.JWKS, true
	case "certificates":
		return bundle.Certificates, true
	case "runner":
		return bundle.Runner, true
	case "artifacts":
		return bundle.Artifacts, true
	case "extra":
		return bundle.Extra, true
	default:
		if bundle.Extra == nil {
			return nil, false
		}
		value, ok := bundle.Extra[name]
		return value, ok
	}
}

func descend(value any, parts []string) (any, bool) {
	current := value
	for _, part := range parts {
		if isNilValue(current) {
			return nil, false
		}
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	if isNilValue(current) {
		return nil, false
	}
	return current, true
}

func isNilValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func typeName(value any) string {
	if value == nil {
		return ""
	}
	t := reflect.TypeOf(value)
	if t == nil {
		return ""
	}
	return t.String()
}
