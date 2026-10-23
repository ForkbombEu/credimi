// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// Status shapes the probe can tell apart. The first ten are malformed
// status-reference shapes the capture issuer emits when asked
// (`status_reference`); `status_absent` is the default issuance with
// `status_list_enabled: false`, whose token carries no status element at all.
//
// The five `uri_*` shapes name one RFC 3986 defect each, so a token retained
// with a specific defect can be attributed to the issuance that carried it: one
// fixture may be issued several times on a device, and only the exact defect
// tells the malformed copy from a valid duplicate.
const (
	statusShapeMissingStatusList         = "missing_status_list"
	statusShapeNegativeIndex             = "negative_index"
	statusShapeMissingIndex              = "missing_index"
	statusShapeMalformedURI              = "malformed_uri"
	statusShapeMissingURI                = "missing_uri"
	statusShapeURIMissingScheme          = "uri_missing_scheme"
	statusShapeURISpaceInHost            = "uri_space_in_host"
	statusShapeURIInvalidPercentEncoding = "uri_invalid_percent_encoding"
	statusShapeURIIllegalCharacters      = "uri_illegal_characters"
	statusShapeURIEmpty                  = "uri_empty"
	statusShapeStatusAbsent              = "status_absent"
)

// OID4VPMalformedStatusCredentialAbsentValidator proves that a Wallet did not
// keep a Referenced Token whose status reference is malformed.
//
// Rejection cannot be observed directly, so it is probed: the query pins the
// document_number of the claim-set fixture the malformed token was issued on
// and asks for every match. A Wallet that rejected the token returns nothing.
// The shape check matters because one fixture may legitimately be issued more
// than once on the same device — the Wallet then holds a valid credential with
// the same document_number, and only the status claim distinguishes it from the
// one that had to be refused.
//
// `vct` selects an SD-JWT VC query, whose status is the JOSE `status` claim;
// `doctype` with `namespace` selects an ISO mdoc query, whose status is the
// `status` element of the Mobile Security Object.
type OID4VPMalformedStatusCredentialAbsentValidator struct{}

// ID returns the validator identifier.
func (OID4VPMalformedStatusCredentialAbsentValidator) ID() string {
	return "oid4vp.malformed_status_credential_absent"
}

type malformedStatusParams struct {
	VCT       string `json:"vct"`
	DocType   string `json:"doctype"`
	Namespace string `json:"namespace"`
	Claim     string `json:"claim"`
	Value     any    `json:"value"`
	Shape     string `json:"shape"`
}

func (p malformedStatusParams) definitionError() string {
	if (p.VCT == "") == (p.DocType == "") {
		return "exactly one of vct or doctype is required"
	}
	if p.DocType != "" && p.Namespace == "" {
		return "namespace param is required with doctype"
	}
	if p.Claim == "" {
		return claimParamRequired
	}
	if p.Value == nil {
		return "value param is required"
	}
	switch p.Shape {
	case statusShapeMissingStatusList,
		statusShapeNegativeIndex,
		statusShapeMissingIndex,
		statusShapeMalformedURI,
		statusShapeMissingURI,
		statusShapeURIMissingScheme,
		statusShapeURISpaceInHost,
		statusShapeURIInvalidPercentEncoding,
		statusShapeURIIllegalCharacters,
		statusShapeURIEmpty,
		statusShapeStatusAbsent:
		return ""
	default:
		return "shape must be missing_status_list, negative_index, missing_index, " +
			"malformed_uri, missing_uri, uri_missing_scheme, uri_space_in_host, " +
			"uri_invalid_percent_encoding, uri_illegal_characters, uri_empty " +
			"or status_absent"
	}
}

// claimPath is the DCQL path the probe must pin.
func (p malformedStatusParams) claimPath() []any {
	if p.DocType != "" {
		return []any{p.Namespace, p.Claim}
	}
	return []any{p.Claim}
}

// Validate requires the pinned, multiple-enabled query to return no
// presentation carrying the malformed status shape.
func (OID4VPMalformedStatusCredentialAbsentValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	return guardWalletEngagement(input.Value, validateMalformedStatusCredentialAbsent(input))
}

func validateMalformedStatusCredentialAbsent(input Input) Result {
	params, err := DecodeParams[malformedStatusParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if message := params.definitionError(); message != "" {
		return Result{Status: StatusError, Message: message}
	}

	root, query, result := capturedDCQLQuery(input.Value)
	if result != nil {
		return *result
	}
	var credential map[string]any
	var credentialID string
	if params.DocType != "" {
		credential, credentialID, result = credentialQueryForDocType(query, params.DocType)
	} else {
		credential, credentialID, result = credentialQueryForVCT(query, params.VCT)
	}
	if result != nil {
		return *result
	}
	if multiple, ok := credential["multiple"].(bool); !ok || !multiple {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"credential query %q does not set multiple to true, "+
					"so a retained malformed credential could stay hidden",
				credentialID,
			),
		}
	}
	if result := requireClaimValueRestriction(
		[]any{credential},
		params.claimPath(),
		params.Value,
	); result != nil {
		return *result
	}

	none := Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"wallet returned no presentation for %s %v",
			params.Claim,
			params.Value,
		),
	}
	responseValue, found := findObjectKey(root, "vp_token")
	if !found {
		return none
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object"}
	}
	entries, _ := response[credentialID].([]any)
	if len(entries) == 0 {
		return none
	}

	for index, entry := range entries {
		malformed, err := presentedStatusIsMalformed(entry, params)
		if err != nil {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("vp_token[%q][%d]: %v", credentialID, index, err),
			}
		}
		if malformed {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"wallet retained a credential with %s %v whose status is %s",
					params.Claim,
					params.Value,
					params.Shape,
				),
			}
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"none of the %d credential(s) with %s %v carries a %s status reference",
			len(entries),
			params.Claim,
			params.Value,
			params.Shape,
		),
	}
}

// presentedStatusIsMalformed decodes one vp_token entry in the query's format
// and reports whether its status carries the probed defect.
func presentedStatusIsMalformed(entry any, params malformedStatusParams) (bool, error) {
	token, ok := entry.(string)
	if !ok || token == "" {
		return false, fmt.Errorf("entry is not a presentation string")
	}
	if params.DocType == "" {
		presentation, err := evidence.ParseSDJWTPresentation(token)
		if err != nil {
			return false, fmt.Errorf("not a valid SD-JWT presentation: %w", err)
		}
		return hasMalformedStatusShape(presentation.Claims, params.Shape), nil
	}
	presentation, err := evidence.ParseMDocPresentation(token)
	if err != nil {
		return false, fmt.Errorf("not a valid mdoc DeviceResponse: %w", err)
	}
	document, ok := presentation.Document(params.DocType)
	if !ok {
		return false, fmt.Errorf("DeviceResponse holds no %q document", params.DocType)
	}
	return hasMalformedMDocStatusShape(document.MSOStatus, params.Shape), nil
}

// credentialQueryForDocType returns the credential query selecting the given
// mdoc doctype together with its id.
func credentialQueryForDocType(
	query map[string]any,
	docType string,
) (map[string]any, string, *Result) {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return nil, "", &Result{
			Status:  StatusFail,
			Message: "dcql_query does not contain credentials",
		}
	}
	for _, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			continue
		}
		meta, ok := normalizeJSONObject(credential["meta"])
		if !ok || meta["doctype_value"] != docType {
			continue
		}
		id, ok := credential["id"].(string)
		if !ok || id == "" {
			return nil, "", &Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("credential query for doctype %q has no id", docType),
			}
		}
		return credential, id, nil
	}
	return nil, "", &Result{
		Status:  StatusFail,
		Message: fmt.Sprintf("dcql_query has no credential query for doctype %q", docType),
	}
}

// hasMalformedStatusShape reports whether the presented claims exhibit the
// defect the issuer was asked to emit. The `uri_*` shapes read
// `status_list.uri` through analyzeStatusURI, so each one names a single RFC
// 3986 defect rather than "somehow invalid".
func hasMalformedStatusShape(claims map[string]any, shape string) bool {
	status, found := resolveObjectPath(claims, "status")
	if !found {
		return shape == statusShapeStatusAbsent
	}
	if _, ok := normalizeJSONObject(status); !ok {
		return false
	}
	statusList, listFound := resolveObjectPath(claims, "status.status_list")
	list, listIsObject := normalizeJSONObject(statusList)
	if shape == statusShapeMissingStatusList {
		return !listFound
	}
	if !listFound || !listIsObject {
		return false
	}

	switch shape {
	case statusShapeMissingIndex:
		_, indexFound := list["idx"]
		return !indexFound
	case statusShapeNegativeIndex:
		index, ok := claimPathIndex(list["idx"])
		return ok && index < 0
	case statusShapeMissingURI:
		_, uriFound := list["uri"]
		return !uriFound
	case statusShapeMalformedURI:
		text, ok := list["uri"].(string)
		if !ok {
			return true
		}
		return analyzeStatusURI(text).malformed()
	case statusShapeURIMissingScheme,
		statusShapeURISpaceInHost,
		statusShapeURIInvalidPercentEncoding,
		statusShapeURIIllegalCharacters,
		statusShapeURIEmpty:
		text, ok := list["uri"].(string)
		return ok && matchesStatusURIShape(text, shape)
	}
	return false
}

// hasMalformedMDocStatusShape is the Mobile Security Object counterpart of
// hasMalformedStatusShape: it reads the CBOR `status` map, so a negative index
// is a CBOR negative integer (major type 1) rather than a JSON number.
func hasMalformedMDocStatusShape(status *evidence.MDocCBORValue, shape string) bool {
	if status == nil {
		return shape == statusShapeStatusAbsent
	}
	if status.ContentMajorType != 5 {
		return false
	}
	list, listFound := status.Member("status_list")
	if shape == statusShapeMissingStatusList {
		return !listFound
	}
	if !listFound || list.ContentMajorType != 5 {
		return false
	}

	switch shape {
	case statusShapeMissingIndex:
		_, indexFound := list.Member("idx")
		return !indexFound
	case statusShapeNegativeIndex:
		index, indexFound := list.Member("idx")
		return indexFound && index.MajorType == 1
	case statusShapeMissingURI:
		_, uriFound := list.Member("uri")
		return !uriFound
	case statusShapeMalformedURI:
		uri, uriFound := list.Member("uri")
		if !uriFound {
			return false
		}
		text, ok := uri.Value.(string)
		if uri.MajorType != 3 || !ok {
			return true
		}
		return analyzeStatusURI(text).malformed()
	case statusShapeURIMissingScheme,
		statusShapeURISpaceInHost,
		statusShapeURIInvalidPercentEncoding,
		statusShapeURIIllegalCharacters,
		statusShapeURIEmpty:
		uri, uriFound := list.Member("uri")
		if !uriFound || uri.MajorType != 3 {
			return false
		}
		text, ok := uri.Value.(string)
		return ok && matchesStatusURIShape(text, shape)
	}
	return false
}

// statusURIDefects lists the RFC 3986 defects a status-list URI carries. One
// URI can carry several at once (a space in the host is also a character
// outside the permitted set), so each probe shape reads its own field instead
// of switching on a single classification.
type statusURIDefects struct {
	empty                  bool
	missingScheme          bool
	spaceInHost            bool
	invalidPercentEncoding bool
	// illegalInPath covers the path, query and fragment: the parts the
	// `uri_illegal_characters` shape names. The authority is checked separately
	// because a space there is the `uri_space_in_host` shape instead.
	illegalInPath bool
	// illegalAnywhere covers the whole URI, which is what `malformed_uri`
	// rejects.
	illegalAnywhere bool
	noHost          bool
	unparseable     bool
}

// malformed reports whether the URI is not a well-formed absolute RFC 3986 URI
// with an authority host.
func (d statusURIDefects) malformed() bool {
	return d.empty || d.missingScheme || d.noHost ||
		d.invalidPercentEncoding || d.illegalAnywhere || d.unparseable
}

// analyzeStatusURI decomposes raw into scheme, authority and remainder and
// records every defect the probe can name. An empty URI reports only `empty`,
// so it never doubles as a missing scheme.
func analyzeStatusURI(raw string) statusURIDefects {
	var defects statusURIDefects
	if raw == "" {
		defects.empty = true
		return defects
	}

	schemeEnd, hasScheme := statusURISchemeEnd(raw)
	defects.missingScheme = !hasScheme
	defects.invalidPercentEncoding = !statusURIPercentEncodingValid(raw)
	defects.illegalAnywhere = statusURIHasIllegalCharacter(raw)

	authorityStart := -1
	switch {
	case hasScheme && strings.HasPrefix(raw[schemeEnd:], "//"):
		authorityStart = schemeEnd + len("//")
	case !hasScheme && strings.HasPrefix(raw, "//"):
		authorityStart = len("//")
	}
	if authorityStart < 0 {
		defects.noHost = true
		remainder := raw
		if hasScheme {
			remainder = raw[schemeEnd:]
		}
		defects.illegalInPath = statusURIHasIllegalCharacter(remainder)
	} else {
		authorityEnd := len(raw)
		if delimiter := strings.IndexAny(raw[authorityStart:], "/?#"); delimiter >= 0 {
			authorityEnd = authorityStart + delimiter
		}
		host := statusURIHost(raw[authorityStart:authorityEnd])
		defects.noHost = host == ""
		defects.spaceInHost = strings.ContainsRune(host, ' ')
		defects.illegalInPath = statusURIHasIllegalCharacter(raw[authorityEnd:])
	}

	if _, err := url.Parse(raw); err != nil {
		defects.unparseable = true
	}
	return defects
}

// matchesStatusURIShape reports whether raw exhibits the named `uri_*` shape.
// Shapes outside that family report false.
func matchesStatusURIShape(raw, shape string) bool {
	defects := analyzeStatusURI(raw)
	switch shape {
	case statusShapeURIMissingScheme:
		return defects.missingScheme
	case statusShapeURISpaceInHost:
		return defects.spaceInHost
	case statusShapeURIInvalidPercentEncoding:
		return defects.invalidPercentEncoding
	case statusShapeURIIllegalCharacters:
		return defects.illegalInPath
	case statusShapeURIEmpty:
		return defects.empty
	}
	return false
}

// statusURISchemeEnd returns the index just past the ':' that terminates an
// RFC 3986 scheme. It reports false when the URI has no scheme, i.e. when an
// ALPHA start is missing or a '/', '?' or '#' arrives before the ':'.
func statusURISchemeEnd(raw string) (int, bool) {
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

// statusURIHost strips any userinfo and port from an authority so a space in
// either is not mistaken for a space in the host.
func statusURIHost(authority string) string {
	if at := strings.LastIndexByte(authority, '@'); at >= 0 {
		authority = authority[at+1:]
	}
	if strings.HasPrefix(authority, "[") {
		if end := strings.IndexByte(authority, ']'); end >= 0 {
			return authority[:end+1]
		}
		return authority
	}
	if colon := strings.IndexByte(authority, ':'); colon >= 0 {
		return authority[:colon]
	}
	return authority
}

// statusURIPercentEncodingValid reports whether every '%' starts a two-digit
// hexadecimal escape.
func statusURIPercentEncodingValid(raw string) bool {
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

// statusURIHasIllegalCharacter reports whether text holds a byte outside RFC
// 3986's permitted set: unreserved, gen-delims, sub-delims and '%'. A space is
// outside it wherever it appears.
func statusURIHasIllegalCharacter(text string) bool {
	for index := range len(text) {
		if !isStatusURIPermittedByte(text[index]) {
			return true
		}
	}
	return false
}

func isStatusURIPermittedByte(character byte) bool {
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
