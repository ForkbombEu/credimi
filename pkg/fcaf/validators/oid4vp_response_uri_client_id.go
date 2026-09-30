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

// OID4VPResponseURIClientIDMismatchValidator proves the precondition of the
// strict Response URI matching rule in OpenID4VP 1.0 Section 5.9: for the
// x509_san_dns Client Identifier Prefix the FQDN of the Response URI must
// equal the Client Identifier without its prefix. It inspects the Request
// Object that was actually delivered to the Wallet and requires the delivered
// response_uri host to differ from that Client Identifier, so the Wallet is
// known to have received a request it must refuse.
type OID4VPResponseURIClientIDMismatchValidator struct{}

func (OID4VPResponseURIClientIDMismatchValidator) ID() string {
	return "oid4vp.response_uri_client_id_mismatch"
}

func (OID4VPResponseURIClientIDMismatchValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		Prefix string `json:"prefix"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	prefix := params.Prefix
	if prefix == "" {
		prefix = "x509_san_dns"
	}

	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	clientID, _ := payload["client_id"].(string)
	if !strings.HasPrefix(clientID, prefix+":") {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"delivered client_id %q does not use the %q Client Identifier Prefix",
				clientID,
				prefix,
			),
		}
	}
	identifier := strings.TrimPrefix(clientID, prefix+":")
	if identifier == "" {
		return Result{Status: StatusFail, Message: "delivered client_id carries no identifier"}
	}

	rawResponseURI, _ := payload["response_uri"].(string)
	if rawResponseURI == "" {
		return Result{Status: StatusFail, Message: "delivered request has no response_uri"}
	}
	responseURI, err := url.Parse(rawResponseURI)
	if err != nil || responseURI.Hostname() == "" {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"delivered response_uri %q is not an absolute URI",
				rawResponseURI,
			),
		}
	}
	if strings.EqualFold(responseURI.Hostname(), identifier) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"delivered response_uri host %q matches the Client Identifier %q",
				responseURI.Hostname(),
				identifier,
			),
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"delivered response_uri host %q does not match the Client Identifier %q",
			responseURI.Hostname(),
			identifier,
		),
	}
}
