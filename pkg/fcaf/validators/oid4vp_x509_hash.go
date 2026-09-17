// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// OID4VPX509HashClientIDValidator verifies the leaf certificate hash binding.
type OID4VPX509HashClientIDValidator struct{}

func (OID4VPX509HashClientIDValidator) ID() string { return "oid4vp.x509_hash_client_id" }

func (OID4VPX509HashClientIDValidator) Validate(_ context.Context, input Input) Result {
	token, ok := input.Value.(string)
	if !ok {
		return Result{Status: StatusFail, Message: "input is not a compact JWS"}
	}
	claims := jwt.MapClaims{}
	parsed, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return Result{Status: StatusFail, Message: fmt.Sprintf("parse Request Object: %v", err)}
	}
	clientID, _ := claims["client_id"].(string)
	const prefix = "x509_hash:"
	if !strings.HasPrefix(clientID, prefix) {
		return Result{Status: StatusFail, Message: "Request Object client_id is not x509_hash"}
	}
	x5c, ok := parsed.Header["x5c"].([]any)
	if !ok || len(x5c) == 0 {
		return Result{Status: StatusFail, Message: "Request Object x5c leaf is missing"}
	}
	encoded, ok := x5c[0].(string)
	if !ok {
		return Result{Status: StatusFail, Message: "Request Object x5c leaf is not a string"}
	}
	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return Result{Status: StatusFail, Message: "Request Object x5c leaf is not base64"}
	}
	if _, err = x509.ParseCertificate(der); err != nil {
		return Result{Status: StatusFail, Message: "Request Object x5c leaf is not a certificate"}
	}
	digest := sha256.Sum256(der)
	if clientID != prefix+base64.RawURLEncoding.EncodeToString(digest[:]) {
		return Result{
			Status:  StatusFail,
			Message: "Request Object client_id does not match its x5c leaf hash",
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "x509_hash client_id matches the x5c leaf certificate",
	}
}
