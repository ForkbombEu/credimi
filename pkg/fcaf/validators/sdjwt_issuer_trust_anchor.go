// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// SDJWTIssuerTrustAnchorExcludedValidator verifies that the presented SD-JWT VC
// carries the issuer signing certificate and its trust chain in `x5c` while the
// trust anchor stays outside that chain.
//
// The observable form of an excluded anchor is that no certificate in `x5c` is
// self-signed and every link is signed by its successor, so the top-most
// certificate is signed by a key the chain does not carry. A trust anchor that
// a trust list designates below a root cannot be distinguished from a regular
// intermediate by inspecting the presentation alone.
type SDJWTIssuerTrustAnchorExcludedValidator struct{}

// ID returns the validator identifier.
func (SDJWTIssuerTrustAnchorExcludedValidator) ID() string {
	return "sdjwt.issuer_trust_anchor_excluded"
}

// Validate walks the presented `x5c` chain and rejects an embedded self-signed
// certificate or a broken issuer link.
func (SDJWTIssuerTrustAnchorExcludedValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	headers, ok := sdjwtProtectedHeaders(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "SD-JWT issuer protected headers are missing"}
	}
	rawChain, ok := headers["x5c"].([]any)
	if !ok || len(rawChain) == 0 {
		return Result{Status: StatusFail, Message: "SD-JWT issuer x5c chain is missing"}
	}
	chain := make([]*x509.Certificate, 0, len(rawChain))
	for index, rawCertificate := range rawChain {
		encoded, ok := rawCertificate.(string)
		if !ok || encoded == "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("SD-JWT issuer x5c certificate %d is not a string", index),
			}
		}
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"SD-JWT issuer x5c certificate %d is not base64 encoded",
					index,
				),
			}
		}
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("SD-JWT issuer x5c certificate %d is not valid DER", index),
			}
		}
		chain = append(chain, certificate)
	}

	for index, certificate := range chain {
		if isSelfSignedCertificate(certificate) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"SD-JWT issuer x5c certificate %d is self-signed, so the trust anchor is included",
					index,
				),
			}
		}
	}
	for index := 0; index+1 < len(chain); index++ {
		child, parent := chain[index], chain[index+1]
		if !bytes.Equal(child.RawIssuer, parent.RawSubject) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"SD-JWT issuer x5c certificate %d is not issued by certificate %d",
					index,
					index+1,
				),
			}
		}
		if err := parent.CheckSignature(
			child.SignatureAlgorithm,
			child.RawTBSCertificate,
			child.Signature,
		); err != nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"SD-JWT issuer x5c certificate %d is not signed by certificate %d: %v",
					index,
					index+1,
					err,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"SD-JWT issuer x5c carries %d certificate(s) signed by a trust anchor it does not include",
			len(chain),
		),
	}
}

// isSelfSignedCertificate reports a certificate that signs itself. The
// signature is checked directly instead of through CheckSignatureFrom, which
// additionally requires the CA basic constraint and would miss a self-signed
// end-entity certificate.
func isSelfSignedCertificate(certificate *x509.Certificate) bool {
	if !bytes.Equal(certificate.RawSubject, certificate.RawIssuer) {
		return false
	}
	return certificate.CheckSignature(
		certificate.SignatureAlgorithm,
		certificate.RawTBSCertificate,
		certificate.Signature,
	) == nil
}
