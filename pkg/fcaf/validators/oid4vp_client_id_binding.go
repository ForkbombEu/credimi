// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// OID4VPX509ClientIDLeafMismatchValidator proves that an X.509 Client
// Identifier does not bind to the leaf certificate delivered in x5c
// (OpenID4VP Section 5.9.3): for x509_hash the value is not the leaf hash, and
// for x509_san_dns the DNS name is not a dNSName SAN of the leaf.
type OID4VPX509ClientIDLeafMismatchValidator struct{}

func (OID4VPX509ClientIDLeafMismatchValidator) ID() string {
	return "oid4vp.x509_client_id_leaf_mismatch"
}

func (OID4VPX509ClientIDLeafMismatchValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Prefix string `json:"prefix"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Prefix != "x509_hash" && params.Prefix != "x509_san_dns" {
		return Result{
			Status:  StatusError,
			Message: "prefix param must be x509_hash or x509_san_dns",
		}
	}

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
	value, found := strings.CutPrefix(clientID, params.Prefix+":")
	if !found || value == "" {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"Request Object client_id does not use the %s prefix",
				params.Prefix,
			),
		}
	}
	leaf, err := requestObjectLeafCertificate(parsed)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	if params.Prefix == "x509_hash" {
		digest := sha256.Sum256(leaf.Raw)
		if value == base64.RawURLEncoding.EncodeToString(digest[:]) {
			return Result{
				Status:  StatusFail,
				Message: "x509_hash client_id matches the x5c leaf certificate",
			}
		}
		return Result{
			Status:  StatusPass,
			Message: "x509_hash client_id is not the hash of the x5c leaf certificate",
		}
	}
	if slices.Contains(leaf.DNSNames, value) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("x5c leaf certificate lists dNSName %q", value),
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"x509_san_dns name %q is not among the x5c leaf dNSName SANs %v",
			value,
			leaf.DNSNames,
		),
	}
}

func requestObjectLeafCertificate(parsed *jwt.Token) (*x509.Certificate, error) {
	x5c, ok := parsed.Header["x5c"].([]any)
	if !ok || len(x5c) == 0 {
		return nil, errors.New("request object x5c leaf is missing")
	}
	encoded, ok := x5c[0].(string)
	if !ok {
		return nil, errors.New("request object x5c leaf is not a string")
	}
	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("request object x5c leaf is not base64")
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, errors.New("request object x5c leaf is not a certificate")
	}
	return leaf, nil
}

// OID4VPDIDSigningKeyUnlistedValidator proves that a decentralized_identifier
// Request Object is signed by a key that none of the verification methods in
// the resolved DID document verifies.
type OID4VPDIDSigningKeyUnlistedValidator struct{}

func (OID4VPDIDSigningKeyUnlistedValidator) ID() string {
	return "oid4vp.did_signing_key_unlisted"
}

func (OID4VPDIDSigningKeyUnlistedValidator) Validate(_ context.Context, input Input) Result {
	evidence, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "DID request evidence is not an object"}
	}
	request, ok := evidence["request_object"].(string)
	if !ok {
		return Result{Status: StatusFail, Message: "DID Request Object is missing"}
	}
	document, ok := normalizeJSONObject(evidence["did_document"])
	if !ok {
		return Result{Status: StatusFail, Message: "DID document is missing"}
	}
	claims := jwt.MapClaims{}
	parsed, _, err := new(jwt.Parser).ParseUnverified(request, claims)
	if err != nil {
		return Result{Status: StatusFail, Message: fmt.Sprintf("parse Request Object: %v", err)}
	}
	// DID verification methods here are P-256 keys; any other algorithm would
	// fail verification for a reason other than the signing key.
	if parsed.Method.Alg() != jwt.SigningMethodES256.Alg() {
		return Result{Status: StatusFail, Message: "Request Object is not signed with ES256"}
	}
	clientID, _ := claims["client_id"].(string)
	if !strings.HasPrefix(clientID, "decentralized_identifier:did:") {
		return Result{
			Status:  StatusFail,
			Message: "Request Object client_id does not use decentralized_identifier:did:",
		}
	}
	methods, _ := document["verificationMethod"].([]any)
	if len(methods) == 0 {
		return Result{Status: StatusFail, Message: "DID document lists no verificationMethod"}
	}

	for _, rawMethod := range methods {
		method, ok := normalizeJSONObject(rawMethod)
		if !ok {
			return Result{Status: StatusFail, Message: "DID verificationMethod is not an object"}
		}
		key, err := didMethodP256Key(method)
		if err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		_, err = jwt.Parse(request, func(*jwt.Token) (any, error) { return key, nil },
			jwt.WithValidMethods([]string{"ES256"}), jwt.WithoutClaimsValidation())
		if err == nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"DID verification method %v verifies the Request Object",
					method["id"],
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"none of the %d DID verification methods verifies the Request Object signature",
			len(methods),
		),
	}
}
