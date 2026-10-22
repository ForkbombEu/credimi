// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// Certificate-chain shapes the capture verifier can deliver.
const (
	chainShapeSelfSignedLeaf = "self_signed_leaf"
	chainShapeUntrustedRoot  = "untrusted_root"
	chainShapeIncomplete     = "incomplete"
	// chainShapeTrustAnchorIncluded is the deployed leaf followed by the CA
	// that issued it (request_behavior.certificate_chain: with_trust_anchor):
	// the request is otherwise valid, so the anchor inside x5c is the only
	// thing a Wallet can object to (WS_RP_SM_RpIntegrity__025).
	chainShapeTrustAnchorIncluded = "trust_anchor_included"
)

// OID4VPRequestCertificateChainValidator proves which defect the delivered
// Request Object's x5c chain actually carries.
//
// The capture verifier signs these requests with the replaced chain's own leaf
// key and recomputes the Client Identifier from it, so the signature verifies
// and the hash matches: the chain is the only thing wrong. A test that asserted
// only "the Wallet rejected it" could not tell these three cases apart, nor
// distinguish them from a corrupted signature.
type OID4VPRequestCertificateChainValidator struct{}

// ID returns the validator identifier.
func (OID4VPRequestCertificateChainValidator) ID() string {
	return "oid4vp.request_certificate_chain"
}

// Validate decodes the delivered x5c chain and requires the named shape.
func (OID4VPRequestCertificateChainValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Shape string `json:"shape"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	switch params.Shape {
	case chainShapeSelfSignedLeaf,
		chainShapeUntrustedRoot,
		chainShapeIncomplete,
		chainShapeTrustAnchorIncluded:
	default:
		return Result{
			Status: StatusError,
			Message: "shape must be self_signed_leaf, untrusted_root, incomplete " +
				"or trust_anchor_included",
		}
	}

	chain, result := deliveredRequestChain(input.Value)
	if result != nil {
		return *result
	}

	switch params.Shape {
	case chainShapeSelfSignedLeaf:
		if len(chain) != 1 {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"delivered x5c carries %d certificate(s), expected a single self-signed leaf",
					len(chain),
				),
			}
		}
		if !isSelfSignedCertificate(chain[0]) {
			return Result{
				Status:  StatusFail,
				Message: "delivered x5c leaf certificate is not self-signed",
			}
		}
		return Result{
			Status:  StatusPass,
			Message: "delivered x5c is a single self-signed certificate",
		}
	case chainShapeIncomplete:
		if result := requireLinkedChain(chain); result != nil {
			return *result
		}
		top := chain[len(chain)-1]
		if isSelfSignedCertificate(top) {
			return Result{
				Status:  StatusFail,
				Message: "delivered x5c terminates in a self-signed certificate, so it is complete",
			}
		}
		return Result{
			Status: StatusPass,
			Message: fmt.Sprintf(
				"delivered x5c carries %d certificate(s) and omits the issuer of its top-most one",
				len(chain),
			),
		}
	case chainShapeTrustAnchorIncluded:
		if len(chain) != 2 {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"delivered x5c carries %d certificate(s), expected the leaf and its trust anchor",
					len(chain),
				),
			}
		}
		if result := requireLinkedChain(chain); result != nil {
			return *result
		}
		if !isSelfSignedCertificate(chain[1]) {
			return Result{
				Status:  StatusFail,
				Message: "delivered x5c[1] is not self-signed, so it is not a trust anchor",
			}
		}
		if isSelfSignedCertificate(chain[0]) {
			return Result{
				Status:  StatusFail,
				Message: "delivered x5c[0] is itself self-signed, so x5c[1] did not issue it",
			}
		}
		return Result{
			Status: StatusPass,
			Message: fmt.Sprintf(
				"delivered x5c is the leaf %q followed by its self-signed issuer %q",
				chain[0].Subject.String(),
				chain[1].Subject.String(),
			),
		}
	default:
		if len(chain) < 2 {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"delivered x5c carries %d certificate(s), expected a leaf and a root",
					len(chain),
				),
			}
		}
		if result := requireLinkedChain(chain); result != nil {
			return *result
		}
		top := chain[len(chain)-1]
		if !isSelfSignedCertificate(top) {
			return Result{
				Status:  StatusFail,
				Message: "delivered x5c does not terminate in a self-signed root",
			}
		}
		if isSelfSignedCertificate(chain[0]) {
			return Result{
				Status:  StatusFail,
				Message: "delivered x5c leaf is itself self-signed, which is the single-certificate shape",
			}
		}
		return Result{
			Status: StatusPass,
			Message: fmt.Sprintf(
				"delivered x5c carries %d certificate(s) terminating in an included self-signed root",
				len(chain),
			),
		}
	}
}

// requireLinkedChain requires every certificate to be signed by its successor,
// so the shape assertions describe one chain rather than unrelated
// certificates that happen to sit in the same header.
func requireLinkedChain(chain []*x509.Certificate) *Result {
	for index := 0; index+1 < len(chain); index++ {
		child, parent := chain[index], chain[index+1]
		if err := parent.CheckSignature(
			child.SignatureAlgorithm,
			child.RawTBSCertificate,
			child.Signature,
		); err != nil {
			return &Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"delivered x5c certificate %d is not signed by certificate %d: %v",
					index,
					index+1,
					err,
				),
			}
		}
	}
	return nil
}

// deliveredRequestChain decodes the x5c header of a compact Request Object.
func deliveredRequestChain(value any) ([]*x509.Certificate, *Result) {
	token, ok := value.(string)
	if !ok {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected a compact Request Object", value),
		}
	}
	parsed, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return nil, &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("parse Request Object: %v", err),
		}
	}
	raw, ok := parsed.Header["x5c"].([]any)
	if !ok || len(raw) == 0 {
		return nil, &Result{
			Status:  StatusFail,
			Message: "delivered Request Object carries no x5c certificate chain",
		}
	}
	chain := make([]*x509.Certificate, 0, len(raw))
	for index, entry := range raw {
		encoded, ok := entry.(string)
		if !ok {
			return nil, &Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("delivered x5c entry %d is not a string", index),
			}
		}
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, &Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("delivered x5c entry %d is not base64: %v", index, err),
			}
		}
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, &Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("delivered x5c entry %d is not a certificate: %v", index, err),
			}
		}
		chain = append(chain, certificate)
	}
	return chain, nil
}
