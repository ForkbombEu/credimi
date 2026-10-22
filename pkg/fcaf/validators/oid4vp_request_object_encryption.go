// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const (
	eventRequestObjectEncrypted             = "vp_request_object_encrypted"
	eventRequestObjectEncryptionUnsupported = "vp_request_object_encryption_unsupported"

	authorizationEncryptionAlgMember = "authorization_encryption_alg_values_supported"
	authorizationEncryptionEncMember = "authorization_encryption_enc_values_supported"
	requestObjectEncryptionAlgMember = "request_object_encryption_alg_values_supported"
	requestObjectEncryptionEncMember = "request_object_encryption_enc_values_supported"
)

// OID4VPEncryptedRequestObjectProcessedValidator applies OpenID4VP 1.0
// Section 5.10 to a Wallet that requires an encrypted Request Object: the
// Wallet advertises public encryption keys and the key-management and
// content-encryption algorithms it accepts in its POSTed wallet_metadata, the
// Request Object is delivered as a compact JWE using those algorithms, and the
// interaction proceeds to a presentation.
//
// The session is ground truth for the negotiation because the encrypted JWE in
// raw.authorization_request_jwe alone does not show which algorithms the Wallet
// advertised: the captured vp_request_object_encrypted event records the alg,
// enc, kid, and the metadata member each value came from. A Wallet that
// publishes no jwks does not require the encrypted Request Object at all, which
// is reported not applicable rather than as a pass so an unexercised feature
// cannot inflate the conformance score.
type OID4VPEncryptedRequestObjectProcessedValidator struct{}

func (OID4VPEncryptedRequestObjectProcessedValidator) ID() string {
	return "oid4vp.encrypted_request_object_processed"
}

func (OID4VPEncryptedRequestObjectProcessedValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: sessionNotObjectMessage}
	}

	metadata, result := encryptedRequestMetadata(session)
	if result != nil {
		return *result
	}

	jwks, _ := normalizeJSONObject(metadata["jwks"])
	if keys, _ := jwks["keys"].([]any); len(keys) == 0 {
		return Result{
			Status: StatusNotApplicable,
			Message: "wallet_metadata publishes no jwks, so the Wallet does not require an " +
				"encrypted Request Object",
		}
	}
	for _, member := range []string{authorizationEncryptionAlgMember, authorizationEncryptionEncMember} {
		if !nonEmptyStringArray(metadata[member]) {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("wallet_metadata publishes no %s", member),
			}
		}
	}

	detail, found := sessionEventDetail(session, eventRequestObjectEncrypted)
	if !found {
		message := fmt.Sprintf(
			"capture recorded no %s event, so the encrypted Request Object was not processed",
			eventRequestObjectEncrypted,
		)
		if unsupported, present := sessionEventDetail(
			session,
			eventRequestObjectEncryptionUnsupported,
		); present {
			message += fmt.Sprintf(
				"; the Wallet reported %s %s",
				eventRequestObjectEncryptionUnsupported,
				describeEventDetail(unsupported),
			)
		}
		return Result{Status: StatusFail, Message: message}
	}
	alg, _ := detail["alg"].(string)
	enc, _ := detail["enc"].(string)
	algSource, _ := detail["alg_source"].(string)
	encSource, _ := detail["enc_source"].(string)
	kid, _ := detail["kid"].(string)

	algorithms := encryptionMetadataValues(
		metadata,
		authorizationEncryptionAlgMember,
		requestObjectEncryptionAlgMember,
	)
	if !slices.Contains(algorithms, alg) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"the Request Object was encrypted with alg %q, which the Wallet did not advertise "+
					"(advertised: %s)",
				alg,
				describeEncryptionValues(algorithms),
			),
		}
	}
	encodings := encryptionMetadataValues(
		metadata,
		authorizationEncryptionEncMember,
		requestObjectEncryptionEncMember,
	)
	if !slices.Contains(encodings, enc) {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"the Request Object was encrypted with enc %q, which the Wallet did not advertise "+
					"(advertised: %s)",
				enc,
				describeEncryptionValues(encodings),
			),
		}
	}

	raw, _ := normalizeJSONObject(session["raw"])
	jwe, _ := raw["authorization_request_jwe"].(string)
	if jwe == "" {
		return Result{
			Status:  StatusFail,
			Message: "capture recorded no raw.authorization_request_jwe, so the encrypted Request Object is missing",
		}
	}
	if parts := strings.Split(jwe, "."); len(parts) != 5 {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"raw.authorization_request_jwe is not a compact JWE: %d parts, expected 5",
				len(parts),
			),
		}
	}

	if vpToken, found := findObjectKey(session, "vp_token"); !found || isEmptyDCQLValue(vpToken) {
		return Result{
			Status: StatusFail,
			Message: "the Wallet did not present after receiving the encrypted Request " +
				"Object",
		}
	}

	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"Request Object encrypted with alg %s enc %s (alg_source %s, enc_source %s, kid %s) "+
				"and the Wallet presented",
			alg,
			enc,
			algSource,
			encSource,
			kid,
		),
	}
}

// encryptedRequestMetadata returns the wallet_metadata the Wallet POSTed to the
// Request URI. A Wallet that posted nothing, or posted no wallet_metadata,
// published no encryption keys, so the encrypted-Request-Object requirement
// does not apply to it.
func encryptedRequestMetadata(session map[string]any) (map[string]any, *Result) {
	observed, _ := normalizeJSONObject(session["observed"])
	requestURIPayload, _ := normalizeJSONObject(observed["request_uri_payload"])
	payload, ok := normalizeJSONObject(requestURIPayload["value"])
	if !ok {
		return nil, &Result{
			Status: StatusNotApplicable,
			Message: "the Wallet did not POST to the Request URI, so it does not require an " +
				"encrypted Request Object",
		}
	}
	posted, present := payload["wallet_metadata"]
	if !present {
		return nil, &Result{
			Status: StatusNotApplicable,
			Message: "the Wallet POSTed no wallet_metadata, so it does not require an " +
				"encrypted Request Object",
		}
	}
	metadata, err := decodeWalletMetadata(posted)
	if err != nil {
		return nil, &Result{Status: StatusFail, Message: err.Error()}
	}
	return metadata, nil
}

// sessionEventDetail returns the detail object of the first captured event of
// one type. Capture records the Request Object encryption negotiation there
// rather than on the session itself, so the event is the only evidence of which
// algorithms and metadata members were used.
func sessionEventDetail(session map[string]any, eventType string) (map[string]any, bool) {
	events, ok := session["events"].([]any)
	if !ok {
		return nil, false
	}
	for _, rawEvent := range events {
		event, ok := normalizeJSONObject(rawEvent)
		if !ok {
			continue
		}
		if normalizeString(event["type"]) != eventType {
			continue
		}
		detail, _ := normalizeJSONObject(event["detail"])
		return detail, true
	}
	return nil, false
}

// encryptionMetadataValues unions the algorithm or encoding values a Wallet
// advertised under the wallet_metadata members OpenID4VP 1.0 Section 5.10
// names. Capture may attribute a value to either member group, so a value is
// acceptable when any advertised list carries it.
func encryptionMetadataValues(metadata map[string]any, members ...string) []string {
	values := make([]string, 0, 4)
	for _, member := range members {
		advertised, _ := metadata[member].([]any)
		for _, value := range advertised {
			text, ok := value.(string)
			if !ok || text == "" || slices.Contains(values, text) {
				continue
			}
			values = append(values, text)
		}
	}
	return values
}

func describeEncryptionValues(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

// describeEventDetail renders an event detail for a failure message, keeping
// the Wallet's own report of why it could not use the encryption.
func describeEventDetail(detail map[string]any) string {
	if len(detail) == 0 {
		return "with no detail"
	}
	encoded, err := json.Marshal(detail)
	if err != nil {
		return fmt.Sprintf("%v", detail)
	}
	return string(encoded)
}
