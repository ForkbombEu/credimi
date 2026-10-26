// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// The FCAF shared test cases (docs/fcaf/suts/shared) are families a test
// invokes as one step, "perform all Shared_JSON test cases on ...". Each
// validator here runs one family, or the cases listed in its `cases` param,
// and fails on the first case that fails, naming it.
//
// Shared_JSON runs on the received bytes: a value re-serialised from parsed
// JSON has already lost duplicate member names, invalid UTF-8 and a byte order
// mark. A selected value is taken as bytes when it is a string, or a Capture
// raw_body object ({base64} or {text}).

// sharedCase is one shared test case: its number within the family and check.
type sharedCase struct {
	id    string
	check func([]byte) error
}

var sharedJSONCases = []sharedCase{
	{"01", sharedJSONSyntax},
	{"02", sharedJSONUniqueMembers},
	{"03", sharedJSONUTF8},
}

// sharedCaseError names the shared test case, Shared_<family>_<id>, a value
// failed.
type sharedCaseError struct {
	name string
	err  error
}

func (f sharedCaseError) Error() string { return f.name + ": " + f.err.Error() }

func (f sharedCaseError) Unwrap() error { return f.err }

// runSharedCases runs the selected cases in order and reports the first that
// fails.
func runSharedCases(family string, cases []sharedCase, only []string, data []byte) error {
	for _, item := range cases {
		if len(only) > 0 && !slices.Contains(only, item.id) {
			continue
		}
		if err := item.check(data); err != nil {
			return sharedCaseError{name: "Shared_" + family + "_" + item.id, err: err}
		}
	}
	return nil
}

// sharedJSONSyntax is Shared_JSON_01: the complete input parses as one RFC
// 8259 JSON text.
func sharedJSONSyntax(data []byte) error {
	if !json.Valid(data) {
		return errors.New("the structure is not a valid RFC 8259 JSON text")
	}
	return nil
}

// sharedJSONUniqueMembers is Shared_JSON_02: no object repeats a member name.
// Names are compared after unescaping, as RFC 8259 section 8.3 compares them.
func sharedJSONUniqueMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	type frame struct {
		object    bool
		names     map[string]bool
		expectKey bool
	}
	var stack []*frame
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("the structure cannot be tokenised: %w", err)
		}
		var top *frame
		if len(stack) > 0 {
			top = stack[len(stack)-1]
		}
		if top != nil && top.object && top.expectKey {
			if name, ok := token.(string); ok {
				if top.names[name] {
					return fmt.Errorf("an object has more than one member named %q", name)
				}
				top.names[name] = true
				top.expectKey = false
				continue
			}
		}
		switch token {
		case json.Delim('{'):
			stack = append(stack, &frame{object: true, names: map[string]bool{}, expectKey: true})
			continue
		case json.Delim('['):
			stack = append(stack, &frame{})
			continue
		case json.Delim('}'), json.Delim(']'):
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 && stack[len(stack)-1].object {
			stack[len(stack)-1].expectKey = true
		}
	}
}

// sharedJSONUTF8 is Shared_JSON_03: UTF-8 bytes encoding Unicode code points
// only (no surrogate, also not as an unpaired \u escape), with no byte order
// mark at the beginning.
func sharedJSONUTF8(data []byte) error {
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		return errors.New("the structure starts with a UTF-8 byte order mark")
	}
	if !utf8.Valid(data) {
		return errors.New("the structure is not valid UTF-8")
	}
	if escape, ok := unpairedSurrogateEscape(data); ok {
		return fmt.Errorf(
			"the string escape %s encodes an unpaired surrogate, not a code point",
			escape,
		)
	}
	return nil
}

// unpairedSurrogateEscape finds a \uXXXX escape inside a JSON string that is
// a surrogate without its partner.
func unpairedSurrogateEscape(data []byte) (string, bool) {
	text := string(data)
	inString := false
	for index := 0; index < len(text); index++ {
		switch text[index] {
		case '"':
			inString = !inString
			continue
		case '\\':
		default:
			continue
		}
		if !inString || index+1 >= len(text) {
			continue
		}
		if text[index+1] != 'u' {
			index++
			continue
		}
		first, ok := hexRune(text, index+2)
		if !ok {
			index++
			continue
		}
		escape := text[index : index+6]
		index += 5
		if !utf16.IsSurrogate(first) {
			continue
		}
		if first >= 0xDC00 {
			return escape, true
		}
		if index+6 < len(text) && text[index+1] == '\\' && text[index+2] == 'u' {
			if second, ok := hexRune(text, index+3); ok && second >= 0xDC00 && second <= 0xDFFF {
				index += 6
				continue
			}
		}
		return escape, true
	}
	return "", false
}

// hexRune reads the four hexadecimal digits of a \uXXXX escape. Parsing with
// a 16-bit size bounds the value to a UTF-16 code unit, so it always fits a
// rune.
func hexRune(text string, start int) (rune, bool) {
	if start+4 > len(text) {
		return 0, false
	}
	value, err := strconv.ParseUint(text[start:start+4], 16, 16)
	if err != nil {
		return 0, false
	}
	return rune(value), true
}

// compactParts splits a compact serialization into its parts and requires
// each to be unpadded base64url (RFC 7515 section 2, RFC 4648 section 5)
// without line breaks, whitespace or other characters.
func compactParts(compact string, want int) ([]string, error) {
	parts := strings.Split(compact, ".")
	if len(parts) != want {
		return nil, fmt.Errorf(
			"the value has %d period characters, expected %d",
			len(parts)-1,
			want-1,
		)
	}
	for index, part := range parts {
		for _, character := range part {
			if !isBase64URLCharacter(character) {
				return nil, fmt.Errorf(
					"part %d contains %q, which is not an unpadded base64url character",
					index+1,
					character,
				)
			}
		}
		if len(part)%4 == 1 {
			return nil, fmt.Errorf("part %d has a length no base64url encoding produces", index+1)
		}
	}
	return parts, nil
}

func isBase64URLCharacter(character rune) bool {
	return character >= 'A' && character <= 'Z' ||
		character >= 'a' && character <= 'z' ||
		character >= '0' && character <= '9' ||
		character == '-' || character == '_'
}

// decodeCompactPart base64url-decodes one part of a compact serialization.
func decodeCompactPart(part string, name string) ([]byte, error) {
	data, err := base64.RawURLEncoding.Strict().DecodeString(part)
	if err != nil {
		return nil, fmt.Errorf("the %s does not base64url-decode: %w", name, err)
	}
	return data, nil
}

// decodeCompactObject decodes one part of a compact serialization to a JSON
// object.
func decodeCompactObject(part string, name string) (map[string]any, error) {
	data, err := decodeCompactPart(part, name)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, fmt.Errorf("the %s is not a JSON object", name)
	}
	return object, nil
}

// sharedJWSCases returns the Shared_JWT_JWS checks of a compact JWS. They read
// the value as text, so the JSON checks run on the decoded header and payload
// bytes.
func sharedJWSCases() []sharedCase {
	parts := func(data []byte) ([]string, error) { return compactParts(string(data), 3) }
	jsonPart := func(index int, name string) func([]byte) error {
		return func(data []byte) error {
			split, err := parts(data)
			if err != nil {
				return err
			}
			decoded, err := decodeCompactPart(split[index], name)
			if err != nil {
				return err
			}
			if err := runSharedCases("JSON", sharedJSONCases, nil, decoded); err != nil {
				return fmt.Errorf("the %s fails %w", name, err)
			}
			return nil
		}
	}
	return []sharedCase{
		{"01", func(data []byte) error { _, err := parts(data); return err }},
		{"02", jsonPart(0, "JOSE Header")},
		{"03", func(data []byte) error {
			split, err := parts(data)
			if err != nil {
				return err
			}
			header, err := decodeCompactObject(split[0], "JOSE Header")
			if err != nil {
				return err
			}
			if len(header) == 0 {
				return errors.New("the JOSE Header has no member")
			}
			if _, ok := header["alg"]; !ok {
				return errors.New("the JOSE Header has no alg member")
			}
			return nil
		}},
		{"04", jsonPart(1, "JWS Payload")},
		{"05", func(data []byte) error {
			split, err := parts(data)
			if err != nil {
				return err
			}
			_, err = decodeCompactObject(split[1], "JWT Claims Set")
			return err
		}},
	}
}

// sharedJWECases returns the Shared_JWT_JWE checks of a compact JWE.
func sharedJWECases() []sharedCase {
	parts := func(data []byte) ([]string, error) { return compactParts(string(data), 5) }
	return []sharedCase{
		{"01", func(data []byte) error { _, err := parts(data); return err }},
		{"02", func(data []byte) error {
			split, err := parts(data)
			if err != nil {
				return err
			}
			decoded, err := decodeCompactPart(split[0], "JWE Protected Header")
			if err != nil {
				return err
			}
			if err := runSharedCases("JSON", sharedJSONCases, nil, decoded); err != nil {
				return fmt.Errorf("the JWE Protected Header fails %w", err)
			}
			return nil
		}},
		{"03", func(data []byte) error {
			split, err := parts(data)
			if err != nil {
				return err
			}
			header, err := decodeCompactObject(split[0], "JWE Protected Header")
			if err != nil {
				return err
			}
			if len(header) < 2 {
				return fmt.Errorf(
					"the JOSE Header has %d member(s), expected at least two",
					len(header),
				)
			}
			for _, member := range []string{"alg", "enc"} {
				if _, ok := header[member]; !ok {
					return fmt.Errorf("the JOSE Header has no %s member", member)
				}
			}
			return nil
		}},
	}
}

type sharedParams struct {
	Cases []string `json:"cases"`
}

// sharedValueBytes returns the bytes a selected value stands for: a string as
// is, or a Capture raw_body object's base64 bytes or text.
func sharedValueBytes(item selectedValue) ([]byte, *Result) {
	switch typed := item.value.(type) {
	case string:
		return []byte(typed), nil
	case map[string]any:
		if encoded, ok := typed["base64"].(string); ok {
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, &Result{
					Status:  StatusError,
					Message: fmt.Sprintf("%s base64 does not decode: %v", item.label, err),
				}
			}
			return data, nil
		}
		if text, ok := typed["text"].(string); ok {
			return []byte(text), nil
		}
		if isRedactionMarker(typed) || typed["redacted"] == true {
			return nil, &Result{
				Status:  StatusInconclusive,
				Message: fmt.Sprintf("%s is redacted, so its bytes are not recorded", item.label),
			}
		}
	}
	return nil, &Result{
		Status: StatusFail,
		Message: fmt.Sprintf(
			"%s is a JSON %s, expected a string or a raw body",
			item.label,
			jsonTypeName(item.value),
		),
	}
}

func validateSharedFamily(input Input, family string, cases []sharedCase) Result {
	params, err := DecodeParams[sharedParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	for _, id := range params.Cases {
		if !slices.ContainsFunc(cases, func(item sharedCase) bool { return item.id == id }) {
			return Result{
				Status:  StatusError,
				Message: fmt.Sprintf("Shared_%s has no case %q", family, id),
			}
		}
	}
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	for _, item := range selected {
		data, result := sharedValueBytes(item)
		if result != nil {
			return *result
		}
		if err := runSharedCases(family, cases, params.Cases, data); err != nil {
			return Result{Status: StatusFail, Message: fmt.Sprintf("%s: %v", item.label, err)}
		}
	}
	scope := "every Shared_" + family + " case"
	if len(params.Cases) > 0 {
		scope = "Shared_" + family + " case(s) " + strings.Join(params.Cases, ", ")
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("%d value(s) pass %s", len(selected), scope),
	}
}

// SharedJSONValidator runs the Shared_JSON test cases on received bytes.
type SharedJSONValidator struct{}

func (SharedJSONValidator) ID() string { return "shared.json" }

func (SharedJSONValidator) Validate(_ context.Context, input Input) Result {
	return validateSharedFamily(input, "JSON", sharedJSONCases)
}

// SharedJWTJWSValidator runs the Shared_JWT_JWS test cases on a compact JWS.
type SharedJWTJWSValidator struct{}

func (SharedJWTJWSValidator) ID() string { return "shared.jwt_jws" }

func (SharedJWTJWSValidator) Validate(_ context.Context, input Input) Result {
	return validateSharedFamily(input, "JWT_JWS", sharedJWSCases())
}

// SharedJWTJWEValidator runs the Shared_JWT_JWE test cases on a compact JWE.
type SharedJWTJWEValidator struct{}

func (SharedJWTJWEValidator) ID() string { return "shared.jwt_jwe" }

func (SharedJWTJWEValidator) Validate(_ context.Context, input Input) Result {
	return validateSharedFamily(input, "JWT_JWE", sharedJWECases())
}
