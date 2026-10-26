// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/tools/inflector"
)

// OID4VCIOfferEngagementValidator evaluates what a Wallet did once the
// End-user followed a Credential Offer link from the issuer's engagement page
// (GET /ui/sessions/{id}/engage). Its input is an exchange with
// session_creation (the delivered deeplink), capture_session, session_requests
// and the engagement step's screenshots.
//
// The engagement flow screenshots `wallet-invoked` once the Wallet's screen
// replaced the page, or `wallet-not-invoked` when the page is still shown, i.e.
// the platform did not hand the link's scheme to the Wallet. The source decides
// what a Wallet that was not invoked means (when_not_invoked): the test is not
// applicable to a Wallet without an optional scheme, a failure for a required
// one, and blocked when the invocation is only a precondition.
//
// Before the Wallet is judged, the delivered offer must be the one the test
// sets up: the deeplink scheme and the members present in or absent from
// raw.credential_offer_delivered. Otherwise the result is blocked.
type OID4VCIOfferEngagementValidator struct{}

func (OID4VCIOfferEngagementValidator) ID() string { return "oid4vci.offer_engagement" }

type offerEngagementParams struct {
	Scheme                string        `json:"scheme"`
	OfferPresent          []string      `json:"offer_present"`
	OfferAbsent           []string      `json:"offer_absent"`
	WhenNotInvoked        Status        `json:"when_not_invoked"`
	ExpectRequests        []string      `json:"expect_requests"`
	ForbidSessionRequests bool          `json:"forbid_session_requests"`
	ForbidUnattributed    requestFilter `json:"forbid_unattributed"`
	InvokedScreenshot     string        `json:"invoked_screenshot"`
	NotInvokedScreenshot  string        `json:"not_invoked_screenshot"`
	EngagementPageEvent   string        `json:"engagement_page_event"`
	ObserveSeconds        *float64      `json:"observe_seconds"`
}

const (
	defaultInvokedScreenshot    = "wallet-invoked"
	defaultNotInvokedScreenshot = "wallet-not-invoked"
	defaultEngagementPageEvent  = "credential_offer_engagement_page_served"
)

func (OID4VCIOfferEngagementValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[offerEngagementParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	switch params.WhenNotInvoked {
	case StatusNotApplicable, StatusBlocked, StatusFail:
	default:
		return Result{
			Status:  StatusError,
			Message: "when_not_invoked must be not_applicable, blocked or fail",
		}
	}
	if params.InvokedScreenshot == "" {
		params.InvokedScreenshot = defaultInvokedScreenshot
	}
	if params.NotInvokedScreenshot == "" {
		params.NotInvokedScreenshot = defaultNotInvokedScreenshot
	}
	if params.EngagementPageEvent == "" {
		params.EngagementPageEvent = defaultEngagementPageEvent
	}
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusBlocked, Message: err.Error()}
	}
	if result := checkDeliveredOffer(exchange, params); result != nil {
		return *result
	}
	engaged, result := engagementTime(exchange, params.EngagementPageEvent)
	if result != nil {
		return *result
	}

	names, failure := screenshotFileNames(exchange.object["screenshots"])
	if failure != nil {
		return Result{
			Status:  StatusInconclusive,
			Message: "the engagement step stored no screenshot: " + failure.Message,
		}
	}
	invoked := screenshotNamed(names, params.InvokedScreenshot)
	notInvoked := screenshotNamed(names, params.NotInvokedScreenshot)
	walletRequests, _ := exchange.laterRequests(engaged, nil, requestFilter{})
	switch {
	case invoked == notInvoked:
		return Result{
			Status: StatusInconclusive,
			Message: fmt.Sprintf(
				"the engagement step stored %s, so whether the Wallet was invoked is not evidenced",
				strings.Join(names, ", "),
			),
		}
	case notInvoked && len(walletRequests) > 0:
		return Result{
			Status: StatusInconclusive,
			Message: fmt.Sprintf(
				"the flow saw the engagement page after following the link, yet the Wallet sent %s",
				describeRequests(walletRequests),
			),
		}
	case notInvoked:
		return Result{
			Status: params.WhenNotInvoked,
			Message: fmt.Sprintf(
				"the Wallet was not invoked: after the End-user followed the %s link the engagement "+
					"page was still shown and the session records no Wallet request",
				params.Scheme,
			),
		}
	}

	for _, suffix := range params.ExpectRequests {
		if len(exchange.sessionRequestsEndingIn(suffix)) == 0 {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"the Wallet was invoked but the session records no request ending in %s",
					suffix,
				),
			}
		}
	}
	if params.ForbidSessionRequests || !params.ForbidUnattributed.empty() {
		session, ledger := exchange.laterRequests(engaged, nil, params.ForbidUnattributed)
		if !params.ForbidSessionRequests {
			session = nil
		}
		if len(session)+len(ledger) > 0 {
			message := fmt.Sprintf(
				"after the End-user engaged at %s the Wallet sent %s",
				engaged.Format(time.RFC3339Nano),
				describeRequests(append(session, ledger...)),
			)
			if len(ledger) > 0 {
				message += " (unattributed requests are assumed to be the Wallet's on an otherwise idle deployment)"
			}
			return Result{Status: StatusFail, Message: message}
		}
		if result := requireObservation(exchange, engaged, params.ObserveSeconds); result != nil {
			return *result
		}
	}
	message := "the Wallet was invoked"
	if len(params.ExpectRequests) > 0 {
		message += " and sent requests ending in " + strings.Join(params.ExpectRequests, ", ")
	}
	if params.ForbidSessionRequests {
		message += " and sent no request for the issuance"
	}
	if !params.ForbidUnattributed.empty() {
		message += "; no unattributed request in the window matches the forbidden paths, " +
			"assuming no other client used the deployment"
	}
	return Result{Status: StatusPass, Message: message}
}

// checkDeliveredOffer requires the delivered deeplink and the Credential Offer
// it carries by value to be the ones the test sets up. The deeplink of the
// session-creation response is what the engagement page serves, so its
// `credential_offer` is exactly what the Wallet received, mutated or not
// (raw.credential_offer_delivered is only recorded for a mutated offer).
func checkDeliveredOffer(exchange oid4vciExchange, params offerEngagementParams) *Result {
	blocked := func(format string, args ...any) *Result {
		return &Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				format,
				args...) + "; the test's Credential Offer was not delivered",
		}
	}
	creation, _ := normalizeJSONObject(exchange.object["session_creation"])
	deeplink, _ := creation["deeplink"].(string)
	if params.Scheme != "" &&
		!strings.HasPrefix(strings.ToLower(deeplink), strings.ToLower(params.Scheme)) {
		return blocked("the session deeplink does not use %s", params.Scheme)
	}
	if len(params.OfferPresent)+len(params.OfferAbsent) == 0 {
		return nil
	}
	parsed, err := url.Parse(deeplink)
	if err != nil {
		return blocked("the session deeplink does not parse: %v", err)
	}
	encoded := parsed.Query().Get("credential_offer")
	if encoded == "" {
		return blocked("the session deeplink carries no Credential Offer by value")
	}
	var offer map[string]any
	if err := json.Unmarshal([]byte(encoded), &offer); err != nil || offer == nil {
		return blocked("the deeplink's credential_offer is not a JSON object")
	}
	for _, member := range params.OfferPresent {
		if !memberPresent(offer, member) {
			return blocked("the delivered Credential Offer has no %s", member)
		}
	}
	for _, member := range params.OfferAbsent {
		if _, present := offer[member]; present {
			return blocked("the delivered Credential Offer carries %s", member)
		}
	}
	return nil
}

// engagementTime is when the engagement page was served, which is when the
// End-user could first follow the link.
func engagementTime(exchange oid4vciExchange, eventType string) (time.Time, *Result) {
	at, found, err := firstSessionEventTime(exchange.session, eventType)
	if err != nil || !found {
		return time.Time{}, &Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				"the session records no timed %s event: the engagement page was never served",
				eventType,
			),
		}
	}
	return at, nil
}

func screenshotNamed(names []string, name string) bool {
	wanted := inflector.Snakecase(name)
	for _, stored := range names {
		if strings.Contains(stored, wanted) {
			return true
		}
	}
	return false
}

// OID4VCICredentialRequestBodyValidator checks the Credential Request body as
// received: an unencrypted request must be application/json and pass every
// Shared_JSON case on its exact bytes. An encrypted request (application/jwt
// to an issuer advertising credential_request_encryption) is outside the
// source, which leaves the encrypted case open, so it is not applicable.
type OID4VCICredentialRequestBodyValidator struct{}

func (OID4VCICredentialRequestBodyValidator) ID() string { return "oid4vci.credential_request_body" }

func (OID4VCICredentialRequestBodyValidator) Validate(_ context.Context, input Input) Result {
	exchange, err := decodeOID4VCIExchange(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	requests := exchange.sessionRequestsEndingIn("/credential")
	if len(requests) == 0 {
		return exchange.missingEvidence("the issuance session records no Credential Request")
	}
	for _, request := range requests {
		body, _ := request.raw["raw_body"].(map[string]any)
		if body == nil {
			return Result{
				Status: StatusInconclusive,
				Message: "the Credential Request records no raw_body; only a session created with " +
					"evidence_mode \"full\" records the received bytes",
			}
		}
		declared, _ := body["content_type"].(string)
		mediaType, _, err := mime.ParseMediaType(declared)
		if err != nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"the Credential Request Content-Type %q does not parse",
					declared,
				),
			}
		}
		switch strings.ToLower(mediaType) {
		case "application/json":
			data, result := sharedValueBytes(
				selectedValue{value: body, label: "the Credential Request body"},
			)
			if result != nil {
				return *result
			}
			if err := runSharedCases("JSON", sharedJSONCases, nil, data); err != nil {
				return Result{
					Status:  StatusFail,
					Message: "the Credential Request body fails " + err.Error(),
				}
			}
		case "application/jwt":
			issuerMetadata, known := exchange.metadata["issuer_metadata"]
			if !known {
				return Result{
					Status:  StatusInconclusive,
					Message: "the Credential Request is application/jwt and the exchange has no issuer_metadata",
				}
			}
			if _, advertised := issuerMetadata["credential_request_encryption"]; !advertised {
				return Result{
					Status: StatusFail,
					Message: "the Credential Request is application/jwt, but the issuer advertises no " +
						"credential_request_encryption",
				}
			}
			return Result{
				Status: StatusNotApplicable,
				Message: "the Wallet encrypted the Credential Request (application/jwt), which the issuer " +
					"advertises; the source requires application/json and leaves the encrypted case " +
					"open (TODO in the source)",
			}
		default:
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"the Credential Request is %s, expected application/json",
					mediaType,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"%d Credential Request(s) are application/json and pass every Shared_JSON case",
			len(requests),
		),
	}
}
