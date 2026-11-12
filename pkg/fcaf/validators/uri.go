// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// URISyntaxValidator requires selected strings to be syntactically correct
// RFC 3986 URIs: a scheme, only permitted characters, valid percent-encoding,
// and at most one fragment delimiter.
type URISyntaxValidator struct{}

func (URISyntaxValidator) ID() string { return "uri.syntax" }

func (URISyntaxValidator) Validate(_ context.Context, input Input) Result {
	selected, result := resolveSelection(input)
	if result != nil {
		return *result
	}
	for _, item := range selected {
		text, result := selectedString(item)
		if result != nil {
			return *result
		}
		if problem := rfc3986URIProblem(text); problem != "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("%s %q %s", item.label, text, problem),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("%d value(s) are syntactically correct RFC 3986 URIs", len(selected)),
	}
}

// rfc3986URIProblem names the first RFC 3986 rule a URI breaks, or returns "".
func rfc3986URIProblem(raw string) string {
	if raw == "" {
		return "is empty"
	}
	end, ok := uriSchemeEnd(raw)
	if !ok {
		return "has no RFC 3986 scheme"
	}
	if end == len(raw) {
		return "has an empty hierarchical part"
	}
	if uriHasIllegalCharacter(raw) {
		return "contains a character outside the RFC 3986 set"
	}
	if !uriPercentEncodingValid(raw) {
		return "contains an invalid percent-encoding"
	}
	if strings.Count(raw, "#") > 1 {
		return "contains more than one fragment delimiter"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "does not parse: " + err.Error()
	}
	if parsed.Host == "" && strings.ContainsAny(parsed.Opaque+parsed.Path, "[]") {
		return "uses [ or ] outside an IP literal host"
	}
	return ""
}

// uriSchemeEnd returns the index just past the ':' that terminates an RFC 3986
// scheme. It reports false when the URI has no scheme, i.e. when an ALPHA start
// is missing or a '/', '?' or '#' arrives before the ':'.
func uriSchemeEnd(raw string) (int, bool) {
	if raw == "" || !isASCIIAlpha(raw[0]) {
		return 0, false
	}
	for index := 1; index < len(raw); index++ {
		character := raw[index]
		switch {
		case character == ':':
			return index + 1, true
		case isASCIIAlpha(character),
			character >= '0' && character <= '9',
			character == '+',
			character == '-',
			character == '.':
		default:
			return 0, false
		}
	}
	return 0, false
}

// uriPercentEncodingValid reports whether every '%' starts a two-digit
// hexadecimal escape.
func uriPercentEncodingValid(raw string) bool {
	for index := 0; index < len(raw); index++ {
		if raw[index] != '%' {
			continue
		}
		if index+2 >= len(raw) ||
			!isASCIIHexDigit(raw[index+1]) ||
			!isASCIIHexDigit(raw[index+2]) {
			return false
		}
		index += 2
	}
	return true
}

// uriHasIllegalCharacter reports whether text holds a byte outside RFC 3986's
// permitted set: unreserved, gen-delims, sub-delims and '%'. A space is outside
// it wherever it appears.
func uriHasIllegalCharacter(text string) bool {
	for index := range len(text) {
		if !isURIPermittedByte(text[index]) {
			return true
		}
	}
	return false
}

func isURIPermittedByte(character byte) bool {
	switch {
	case isASCIIAlpha(character), character >= '0' && character <= '9':
		return true
	}
	switch character {
	case '-', '.', '_', '~', // unreserved
		':', '/', '?', '#', '[', ']', '@', // gen-delims
		'!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', // sub-delims
		'%':
		return true
	}
	return false
}

func isASCIIAlpha(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
}

func isASCIIHexDigit(character byte) bool {
	return character >= '0' && character <= '9' ||
		character >= 'a' && character <= 'f' ||
		character >= 'A' && character <= 'F'
}
