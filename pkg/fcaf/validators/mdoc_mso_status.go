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

type mdocMSOStatusParams struct {
	Path []string `json:"path"`
}

// resolveMDocMSOStatus resolves a path of text map keys inside the Mobile
// Security Object `status` value. An empty path addresses `status` itself.
func resolveMDocMSOStatus(
	value any,
	path []string,
) (*evidence.MDocCBORValue, *Result) {
	presentation, ok := mdocPresentation(value)
	if !ok {
		result := wrongMDocInput(value)
		return nil, &result
	}
	status, exists := presentation.SecurityObjectStatus()
	if document, ok := presentation.Document(pidMDocType); ok && document.MSOStatus != nil {
		status, exists = document.MSOStatus, true
	}
	if !exists {
		return nil, &Result{
			Status:  StatusFail,
			Message: "mobile security object has no status element",
		}
	}
	resolved, found := status.Member(path...)
	if !found {
		return nil, &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"mobile security object status has no %q element",
				mdocMSOStatusPathName(path),
			),
		}
	}
	return resolved, nil
}

func mdocMSOStatusPathName(path []string) string {
	if len(path) == 0 {
		return "status"
	}
	return "status." + strings.Join(path, ".")
}

// MDocMSOStatusMemberPresentValidator verifies that the Mobile Security Object
// of the presented mdoc carries the addressed status element.
type MDocMSOStatusMemberPresentValidator struct{}

// ID returns the validator identifier.
func (MDocMSOStatusMemberPresentValidator) ID() string {
	return "mdoc.mso_status_member_present"
}

// Validate requires the addressed status element to be present.
func (MDocMSOStatusMemberPresentValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[mdocMSOStatusParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if _, result := resolveMDocMSOStatus(input.Value, params.Path); result != nil {
		return *result
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"mobile security object contains %q",
			mdocMSOStatusPathName(params.Path),
		),
	}
}

// MDocMSOStatusCBORTypeValidator verifies the CBOR major type of an element
// inside the Mobile Security Object `status` value.
type MDocMSOStatusCBORTypeValidator struct{}

// ID returns the validator identifier.
func (MDocMSOStatusCBORTypeValidator) ID() string { return "mdoc.mso_status_cbor_type" }

// Validate requires the addressed status element to carry the expected CBOR
// major type on the wire.
func (MDocMSOStatusCBORTypeValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Path      []string `json:"path"`
		MajorType uint8    `json:"major_type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if _, exists := input.Params["major_type"]; !exists {
		return Result{Status: StatusError, Message: "major_type param is required"}
	}
	if params.MajorType > 7 {
		return Result{Status: StatusError, Message: "major_type must be between 0 and 7"}
	}
	element, result := resolveMDocMSOStatus(input.Value, params.Path)
	if result != nil {
		return *result
	}
	if element.MajorType != params.MajorType {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"%s has CBOR major type %d, expected %d",
				mdocMSOStatusPathName(params.Path),
				element.MajorType,
				params.MajorType,
			),
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%s has CBOR major type %d",
			mdocMSOStatusPathName(params.Path),
			params.MajorType,
		),
	}
}

// MDocMSOStatusURIValidator verifies that an element inside the Mobile
// Security Object `status` value is a CBOR text string holding an RFC 3986 URI.
type MDocMSOStatusURIValidator struct{}

// ID returns the validator identifier.
func (MDocMSOStatusURIValidator) ID() string { return "mdoc.mso_status_uri" }

// Validate requires the addressed status element to be a text string that
// parses as an absolute URI.
func (MDocMSOStatusURIValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[mdocMSOStatusParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	element, result := resolveMDocMSOStatus(input.Value, params.Path)
	if result != nil {
		return *result
	}
	name := mdocMSOStatusPathName(params.Path)
	if element.MajorType != 3 {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"%s has CBOR major type %d, expected text string type 3",
				name,
				element.MajorType,
			),
		}
	}
	text, ok := element.Value.(string)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("%s is %T, expected a text string", name, element.Value),
		}
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("%s is not a valid URI: %v", name, err),
		}
	}
	if !parsed.IsAbs() {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("%s is not an absolute URI", name),
		}
	}
	return Result{Status: StatusPass, Message: fmt.Sprintf("%s is an RFC 3986 URI", name)}
}
