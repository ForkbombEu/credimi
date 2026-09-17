// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// OID4VPDIDSignedRequestValidator verifies a Request Object with the public
// key selected by its kid from the published DID document.
type OID4VPDIDSignedRequestValidator struct{}

func (OID4VPDIDSignedRequestValidator) ID() string { return "oid4vp.did_signed_request" }

func (OID4VPDIDSignedRequestValidator) Validate(_ context.Context, input Input) Result {
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
	_, err := jwt.Parse(request, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("request object kid is missing")
		}
		methods, ok := document["verificationMethod"].([]any)
		if !ok {
			return nil, fmt.Errorf("did verificationMethod is missing")
		}
		for _, method := range methods {
			method, ok := normalizeJSONObject(method)
			if !ok || method["id"] != kid {
				continue
			}
			jwk, ok := normalizeJSONObject(method["publicKeyJwk"])
			if !ok {
				return nil, fmt.Errorf("did method JWK is missing")
			}
			if jwk["kty"] != "EC" || jwk["crv"] != "P-256" {
				return nil, fmt.Errorf("did method is not P-256 EC")
			}
			x, xok := jwk["x"].(string)
			y, yok := jwk["y"].(string)
			if !xok || !yok {
				return nil, fmt.Errorf("did method EC coordinates are missing")
			}
			xb, err := base64.RawURLEncoding.DecodeString(x)
			if err != nil {
				return nil, fmt.Errorf("decode DID x coordinate: %w", err)
			}
			yb, err := base64.RawURLEncoding.DecodeString(y)
			if err != nil {
				return nil, fmt.Errorf("decode DID y coordinate: %w", err)
			}
			encoded := make([]byte, 1, 1+len(xb)+len(yb))
			encoded[0] = 4
			encoded = append(encoded, xb...)
			encoded = append(encoded, yb...)
			key, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), encoded)
			if err != nil {
				return nil, fmt.Errorf("parse DID method EC key: %w", err)
			}
			return key, nil
		}
		return nil, fmt.Errorf("request object kid is not published by DID document")
	})
	if err != nil {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("DID Request Object verification failed: %v", err),
		}
	}
	return Result{Status: StatusPass, Message: "DID-published key verifies the Request Object"}
}
