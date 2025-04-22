// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestNormalizeProtocolAndAuthor(t *testing.T) {
	tests := []struct {
		name           string
		protocol       string
		author         string
		expectedProto  string
		expectedAuthor string
	}{
		{
			name:           "Normalize openid4vp_wallet protocol",
			protocol:       "openid4vp_wallet",
			author:         "some_author",
			expectedProto:  "OpenID4VP_Wallet",
			expectedAuthor: "some_author",
		},
		{
			name:           "Normalize openid4vci_wallet protocol",
			protocol:       "openid4vci_wallet",
			author:         "some_author",
			expectedProto:  "OpenID4VCI_Wallet",
			expectedAuthor: "some_author",
		},
		{
			name:           "Normalize openid_foundation author",
			protocol:       "some_protocol",
			author:         "openid_foundation",
			expectedProto:  "some_protocol",
			expectedAuthor: "OpenID_foundation",
		},
		{
			name:           "No normalization needed",
			protocol:       "some_protocol",
			author:         "some_author",
			expectedProto:  "some_protocol",
			expectedAuthor: "some_author",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProto, gotAuthor := normalizeProtocolAndAuthor(tt.protocol, tt.author)
			if gotProto != tt.expectedProto {
				t.Errorf("normalizeProtocolAndAuthor() protocol = %v, want %v", gotProto, tt.expectedProto)
			}
			if gotAuthor != tt.expectedAuthor {
				t.Errorf("normalizeProtocolAndAuthor() author = %v, want %v", gotAuthor, tt.expectedAuthor)
			}
		})
	}
}

func TestWriteAPIError(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		domain         string
		reason         string
		message        string
		expectedStatus int
		expectedBody   APIErrorResponse
	}{
		{
			name:           "Basic error",
			code:           400,
			domain:         "testDomain",
			reason:         "invalid",
			message:        "Invalid input",
			expectedStatus: 400,
			expectedBody: APIErrorResponse{
				APIVersion: "2.0",
				Error: APIError{
					Code:    400,
					Message: "Invalid input",
					Errors: []APIErrorDetail{
						{
							Domain:  "testDomain",
							Reason:  "invalid",
							Message: "Invalid input",
						},
					},
				},
			},
		},
		{
			name:           "Different code and message",
			code:           404,
			domain:         "resource",
			reason:         "notFound",
			message:        "Resource not found",
			expectedStatus: 404,
			expectedBody: APIErrorResponse{
				APIVersion: "2.0",
				Error: APIError{
					Code:    404,
					Message: "Resource not found",
					Errors: []APIErrorDetail{
						{
							Domain:  "resource",
							Reason:  "notFound",
							Message: "Resource not found",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			WriteAPIError(rr, tt.code, tt.domain, tt.reason, tt.message)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.expectedStatus)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", ct)
			}

			var got APIErrorResponse
			dec := json.NewDecoder(bytes.NewReader(rr.Body.Bytes()))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&got); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if got.APIVersion != tt.expectedBody.APIVersion {
				t.Errorf("APIVersion = %v, want %v", got.APIVersion, tt.expectedBody.APIVersion)
			}
			if got.Error.Code != tt.expectedBody.Error.Code {
				t.Errorf("Error.Code = %v, want %v", got.Error.Code, tt.expectedBody.Error.Code)
			}
			if got.Error.Message != tt.expectedBody.Error.Message {
				t.Errorf("Error.Message = %v, want %v", got.Error.Message, tt.expectedBody.Error.Message)
			}
			if len(got.Error.Errors) != 1 {
				t.Fatalf("Error.Errors length = %d, want 1", len(got.Error.Errors))
			}
			wantDetail := tt.expectedBody.Error.Errors[0]
			gotDetail := got.Error.Errors[0]
			if gotDetail.Domain != wantDetail.Domain {
				t.Errorf("Error.Errors[0].Domain = %v, want %v", gotDetail.Domain, wantDetail.Domain)
			}
			if gotDetail.Reason != wantDetail.Reason {
				t.Errorf("Error.Errors[0].Reason = %v, want %v", gotDetail.Reason, wantDetail.Reason)
			}
			if gotDetail.Message != wantDetail.Message {
				t.Errorf("Error.Errors[0].Message = %v, want %v", gotDetail.Message, wantDetail.Message)
			}
		})
	}
}

