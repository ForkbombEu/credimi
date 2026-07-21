// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

type Bundle struct {
	RawRequestObject            map[string]any `json:"raw_request_object,omitempty"`
	DecodedRequestObject        map[string]any `json:"decoded_request_object,omitempty"`
	RawPresentationResponse     map[string]any `json:"raw_presentation_response,omitempty"`
	DecodedPresentationResponse map[string]any `json:"decoded_presentation_response,omitempty"`
	VPToken                     any            `json:"vp_token,omitempty"`
	PresentationSubmission      map[string]any `json:"presentation_submission,omitempty"`
	DecodedSDJWT                map[string]any `json:"decoded_sdjwt,omitempty"`
	MDoc                        map[string]any `json:"mdoc,omitempty"`
	IssuerMetadata              map[string]any `json:"issuer_metadata,omitempty"`
	VerifierMetadata            map[string]any `json:"verifier_metadata,omitempty"`
	AuthorizationServerMetadata map[string]any `json:"authorization_server_metadata,omitempty"`
	JWKS                        map[string]any `json:"jwks,omitempty"`
	Certificates                map[string]any `json:"certificates,omitempty"`
	Runner                      map[string]any `json:"runner,omitempty"`
	Artifacts                   map[string]any `json:"artifacts,omitempty"`
	PipelineOutputs             map[string]any `json:"pipeline_outputs,omitempty"`
	Preconditions               map[string]any `json:"preconditions,omitempty"`
	Runtime                     map[string]any `json:"runtime,omitempty"`
	Extra                       map[string]any `json:"extra,omitempty"`
}

type LookupResult struct {
	Path  string `json:"path"`
	Found bool   `json:"found"`
	Value any    `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
}
