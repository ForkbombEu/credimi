// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type SDJWTPresentation struct {
	Raw              string         `json:"raw,omitempty"`
	Claims           map[string]any `json:"claims"`
	ProtectedHeaders map[string]any `json:"protected_headers"`
	IssuerPayload    map[string]any `json:"issuer_payload"`
	KeyBinding       map[string]any `json:"key_binding,omitempty"`
	DisclosureCount  int            `json:"disclosure_count"`
}

func (p *SDJWTPresentation) Claim(name string) (any, bool) {
	if p == nil {
		return nil, false
	}
	current := any(p.Claims)
	for _, segment := range strings.Split(name, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func Extract(root any, path string, decoder string) (any, error) {
	raw, err := extractRaw(root, path)
	if err != nil {
		return nil, err
	}
	return decode(raw, decoder)
}

func extractRaw(root any, path string) (any, error) {
	if !strings.HasPrefix(path, "$.") {
		return nil, fmt.Errorf("invalid pointer %q", path)
	}
	current := root
	remaining := strings.TrimPrefix(path, "$.")
	for remaining != "" {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid pointer segment near %q", remaining)
		}

		key, rest, ok := matchMapKey(obj, remaining)
		if !ok {
			return nil, fmt.Errorf("missing key in path %q", remaining)
		}
		current = obj[key]
		remaining = rest

		for strings.HasPrefix(remaining, "[") {
			end := strings.Index(remaining, "]")
			if end < 0 {
				return nil, fmt.Errorf("invalid pointer index %q", remaining)
			}
			index, err := strconv.Atoi(remaining[1:end])
			if err != nil {
				return nil, fmt.Errorf("invalid pointer index %q", remaining[1:end])
			}
			array, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("segment %q is not an array", key)
			}
			if index < 0 || index >= len(array) {
				return nil, fmt.Errorf("array index %d out of range", index)
			}
			current = array[index]
			remaining = remaining[end+1:]
		}

		remaining = strings.TrimPrefix(remaining, ".")
	}
	return current, nil
}

func matchMapKey(obj map[string]any, remaining string) (string, string, bool) {
	longestKey := ""
	for key := range obj {
		if !strings.HasPrefix(remaining, key) {
			continue
		}
		rest := strings.TrimPrefix(remaining, key)
		if rest != "" && !strings.HasPrefix(rest, ".") && !strings.HasPrefix(rest, "[") {
			continue
		}
		if len(key) > len(longestKey) {
			longestKey = key
		}
	}
	if longestKey == "" {
		return "", "", false
	}
	return longestKey, strings.TrimPrefix(remaining, longestKey), true
}

func decode(raw any, decoder string) (any, error) {
	switch decoder {
	case "", "raw":
		return raw, nil
	case "string":
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("wrong decoder string for %T", raw)
		}
		return value, nil
	case "json":
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("wrong decoder json for %T", raw)
		}
		var out any
		if err := json.Unmarshal([]byte(value), &out); err != nil {
			return nil, fmt.Errorf("decode json: %w", err)
		}
		return out, nil
	case "sdjwt.presentation":
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("wrong decoder sdjwt.presentation for %T", raw)
		}
		return ParseSDJWTPresentation(value)
	case "sdjwt.vp_token_json":
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("wrong decoder sdjwt.vp_token_json for %T", raw)
		}
		token, err := extractPresentationTokenFromVPTokenJSON(value, "query_0")
		if err != nil {
			return nil, err
		}
		return ParseSDJWTPresentation(token)
	case "mdoc.presentation":
		return ParseMDocPresentation(raw)
	case "mdoc.vp_token_json":
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("wrong decoder mdoc.vp_token_json for %T", raw)
		}
		token, err := extractPresentationTokenFromVPTokenJSON(value, "")
		if err != nil {
			return nil, err
		}
		return ParseMDocPresentation(token)
	default:
		return nil, fmt.Errorf("unsupported decoder %q", decoder)
	}
}

func extractPresentationTokenFromVPTokenJSON(raw string, preferredKey string) (string, error) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", fmt.Errorf("decode vp_token json: %w", err)
	}

	key := preferredKey
	if key == "" {
		key = onlyVPTokenCredentialKey(parsed)
	} else if _, ok := parsed[key]; !ok {
		key = onlyVPTokenCredentialKey(parsed)
	}

	if key == "" {
		return "", fmt.Errorf("vp_token json must contain %q or exactly one credential entry", preferredKey)
	}
	rawQuery, ok := parsed[key]
	if !ok {
		return "", fmt.Errorf("vp_token json missing %q", key)
	}

	query, ok := rawQuery.([]any)
	if !ok {
		return "", fmt.Errorf("vp_token %q must be an array", key)
	}
	if len(query) == 0 {
		return "", fmt.Errorf("vp_token %q is empty", key)
	}
	token, ok := query[0].(string)
	if !ok {
		return "", fmt.Errorf("vp_token %q[0] must be a string", key)
	}
	return token, nil
}

func onlyVPTokenCredentialKey(parsed map[string]any) string {
	if len(parsed) != 1 {
		return ""
	}
	for key := range parsed {
		return key
	}
	return ""
}

func ParseSDJWTPresentation(token string) (*SDJWTPresentation, error) {
	segments := strings.Split(token, "~")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid SD-JWT presentation")
	}

	header, payload, err := decodeJWT(strings.TrimSpace(segments[0]))
	if err != nil {
		return nil, err
	}

	if algorithm, _ := payload["_sd_alg"].(string); algorithm != "" && algorithm != "sha-256" {
		return nil, fmt.Errorf("unsupported SD-JWT disclosure digest algorithm %q", algorithm)
	}

	disclosures := map[string]disclosureValue{}
	var keyBinding map[string]any

	for _, segment := range segments[1:] {
		if strings.TrimSpace(segment) == "" {
			continue
		}

		if strings.Count(segment, ".") == 2 {
			kbHeader, kbPayload, err := decodeJWT(segment)
			if err != nil {
				return nil, fmt.Errorf("decode key binding jwt: %w", err)
			}
			kbPayload["_protected_header"] = kbHeader
			keyBinding = kbPayload
			continue
		}

		disclosure, digest, err := parseDisclosure(segment)
		if err != nil {
			return nil, err
		}
		if _, exists := disclosures[digest]; exists {
			return nil, fmt.Errorf("duplicate SD-JWT disclosure")
		}
		disclosures[digest] = disclosure
	}

	used := map[string]struct{}{}
	reconstructed, err := reconstructSDJWTValue(payload, disclosures, used)
	if err != nil {
		return nil, err
	}
	claims, ok := reconstructed.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("SD-JWT issuer payload is not an object")
	}
	delete(claims, "_sd_alg")
	for digest := range disclosures {
		if _, ok := used[digest]; !ok {
			return nil, fmt.Errorf("disclosure digest is not referenced by the SD-JWT")
		}
	}

	return &SDJWTPresentation{
		Raw:              token,
		Claims:           claims,
		ProtectedHeaders: header,
		IssuerPayload:    payload,
		KeyBinding:       keyBinding,
		DisclosureCount:  len(disclosures),
	}, nil
}

type disclosureValue struct {
	Name    string
	Value   any
	IsArray bool
}

func parseDisclosure(segment string) (disclosureValue, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return disclosureValue{}, "", fmt.Errorf("decode disclosure: %w", err)
	}
	if !utf8.Valid(decoded) {
		return disclosureValue{}, "", fmt.Errorf("disclosure is not valid UTF-8")
	}
	var parts []any
	if err := json.Unmarshal(decoded, &parts); err != nil {
		return disclosureValue{}, "", fmt.Errorf("parse disclosure json: %w", err)
	}
	if len(parts) != 2 && len(parts) != 3 {
		return disclosureValue{}, "", fmt.Errorf("disclosure must contain 2 or 3 elements")
	}
	digest := sha256Base64URL(segment)
	if len(parts) == 2 {
		return disclosureValue{Value: parts[1], IsArray: true}, digest, nil
	}
	name, ok := parts[1].(string)
	if !ok {
		return disclosureValue{}, "", fmt.Errorf("disclosure claim name must be a string")
	}
	return disclosureValue{Name: name, Value: parts[2]}, digest, nil
}

func reconstructSDJWTValue(
	value any,
	disclosures map[string]disclosureValue,
	used map[string]struct{},
) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, rawValue := range typed {
			if key == "_sd" {
				digests, ok := rawValue.([]any)
				if !ok {
					return nil, fmt.Errorf("_sd must be an array")
				}
				for _, rawDigest := range digests {
					digest, ok := rawDigest.(string)
					if !ok {
						return nil, fmt.Errorf("_sd digest must be a string")
					}
					disclosure, disclosed := disclosures[digest]
					if !disclosed {
						continue
					}
					if disclosure.IsArray {
						return nil, fmt.Errorf("array disclosure referenced from object _sd")
					}
					if _, duplicate := out[disclosure.Name]; duplicate {
						return nil, fmt.Errorf("duplicate disclosed claim %q", disclosure.Name)
					}
					reconstructed, err := reconstructSDJWTValue(disclosure.Value, disclosures, used)
					if err != nil {
						return nil, err
					}
					out[disclosure.Name] = reconstructed
					used[digest] = struct{}{}
				}
				continue
			}
			reconstructed, err := reconstructSDJWTValue(rawValue, disclosures, used)
			if err != nil {
				return nil, err
			}
			out[key] = reconstructed
		}
		return out, nil
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			placeholder, ok := item.(map[string]any)
			if ok && len(placeholder) == 1 {
				rawDigest, disclosedPlaceholder := placeholder["..."]
				if disclosedPlaceholder {
					digest, ok := rawDigest.(string)
					if !ok {
						return nil, fmt.Errorf("array disclosure digest must be a string")
					}
					disclosure, disclosed := disclosures[digest]
					if !disclosed {
						continue
					}
					if !disclosure.IsArray {
						return nil, fmt.Errorf("object disclosure referenced from array")
					}
					reconstructed, err := reconstructSDJWTValue(disclosure.Value, disclosures, used)
					if err != nil {
						return nil, err
					}
					out = append(out, reconstructed)
					used[digest] = struct{}{}
					continue
				}
			}
			reconstructed, err := reconstructSDJWTValue(item, disclosures, used)
			if err != nil {
				return nil, err
			}
			out = append(out, reconstructed)
		}
		return out, nil
	default:
		return value, nil
	}
}

func decodeJWT(token string) (map[string]any, map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, nil, fmt.Errorf("invalid jwt")
	}

	header, err := decodeBase64JSON(parts[0])
	if err != nil {
		return nil, nil, fmt.Errorf("decode jwt header: %w", err)
	}
	payload, err := decodeBase64JSON(parts[1])
	if err != nil {
		return nil, nil, fmt.Errorf("decode jwt payload: %w", err)
	}
	return header, payload, nil
}

func decodeBase64JSON(part string) (map[string]any, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(decoded, &out); err != nil {
		return nil, err
	}
	return out, nil
}
