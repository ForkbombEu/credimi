// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"
	"go.temporal.io/sdk/testsuite"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UnitTestSuite struct {
        suite.Suite
        testsuite.WorkflowTestSuite

        env *testsuite.TestWorkflowEnvironment
}

func (s *UnitTestSuite) SetupTest() {
        s.env = s.NewTestWorkflowEnvironment()
}

const testDataDir = "../../../test_pb_data"

func generateToken(collectionNameOrId string, email string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	record, err := app.FindAuthRecordByEmail(collectionNameOrId, email)
	if err != nil {
		return "", err
	}

	return record.NewAuthToken()
}

func TestAddOpenID4VPTestEndpoints_RoutesRegistered(t *testing.T) {
	godotenv.Load("../../../.env")

	app, err := tests.NewTestApp(testDataDir)
	defer app.Cleanup()
	require.NoError(t, err)

	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}
		AddComplianceChecks(testApp)

		return testApp
	}

	authToken, err := generateToken("users", "userA@example.org")

	scenarios := []tests.ApiScenario{
		{
			Name:           "OpenID4VP Test - Valid Request",
			Method:         "POST",
			URL:            "/api/compliance/check/OpenID4VP_Wallet/OpenID_Foundation/save-variables-and-start",
			Body:           strings.NewReader(`{"sd_jwt_vc:did:request_uri_signed:direct_post.json":{"format":"variables","data":{"oid_description":{"type":"string","value":"jikj","fieldName":"description"},"oid_alias":{"type":"string","value":"knnkn","fieldName":"testalias"},"oid_client_id":{"type":"string","value":"did:web:app.altme.io:issuer","fieldName":"client_id"},"oid_client_jwks":{"type":"object","value":"{\n    \"keys\": [\n        {\n            \"kty\": \"EC\",\n            \"alg\": \"ES256\",\n            \"crv\": \"P-256\",\n            \"d\": \"GSbo9TpmGaLgxxO6RNx6QnvcfykQJS7vUVgTe8vy9W0\",\n            \"x\": \"m5uKsE35t3sP7gjmirUewufx2Gt2n6J7fSW68apB2Lo\",\n            \"y\": \"-V54TpMI8RbpB40hbAocIjnaHX5WP6NHjWkHfdCSAyU\"\n        }\n    ]\n}","fieldName":"jwks"},"oid_client_presentation_definition":{"type":"object","value":"{\n    \"id\": \"two_sd_jwt\",\n    \"input_descriptors\": [\n        {\n            \"constraints\": {\n                \"fields\": [\n                    {\n                        \"filter\": {\n                            \"const\": \"urn:eu.europa.ec.eudi:pid:1\",\n                            \"type\": \"string\"\n                        },\n                        \"path\": [\n                            \"$.vct\"\n                        ]\n                    }\n                ]\n            },\n            \"format\": {\n                \"vc+sd-jwt\": {\n                    \"kb-jwt_alg_values\": [\n                        \"ES256\",\n                        \"ES256K\",\n                        \"EdDSA\"\n                    ],\n                    \"sd-jwt_alg_values\": [\n                        \"ES256\",\n                        \"ES256K\",\n                        \"EdDSA\"\n                    ]\n                }\n            },\n            \"id\": \"pid_credential\"\n        }\n    ]\n}","fieldName":"presentation_definition"}}}}`),
			Headers:        map[string]string{"Content-Type": "application/json", "Authorization": authToken},
			Delay:          0,
			Timeout:        5 * time.Second,
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"start",
			},
			NotExpectedContent: []string{"error"},
			TestAppFactory:     setupTestApp,
		},
		{
			Name:           "OpenID4VP Test - Invalid Request",
			Method:         "POST",
			URL:            "/api/compliance/check/OpenID4VP_Wallet/OpenID_Foundation/save-variables-and-start",
			Body:           strings.NewReader(`{"sd_jwt_vc:did:request_uri_signed:direct_post.json":{"format":"variables","data":{"oid_description":{"type":"string","value":"jikj","fieldName":"description"},"oid_alias":{"type":"string","value":"knnkn","fieldName":"testalias"},"oid_client_id":{"type":"string","value":"did:web:app.altme.io:issuer","fieldName":"client_id"},"oid_client_jwks":{"type":"object","value":"{\n    \"keys\": [\n        {\n            \"kty\": \"EC\",\n            \"alg\": \"ES256\",\n            \"crv\": \"P-256\",\n            \"d\": \"GSbo9TpmGaLgxxO6RNx6QnvcfykQJS7vUVgTe8vy9W0\",\n            \"x\": \"m5uKsE35t3sP7gjmirUewufx2Gt2n6J7fSW68apB2Lo\",\n            \"y\": \"-V54TpMI8RbpB40hbAocIjnaHX5WP6NHjWkHfdCSAyU\"\n        }\n    ]\n}","fieldName":"jwks"},"oid_client_presentation_definition":{"type":"object","value":"{\n    \"id\": \"two_sd_jwt\",\n    \"input_descriptors\": [\n        {\n            \"constraints\": {\n                \"fields\": [\n                    {\n                        \"filter\": {\n                            \"const\": \"urn:eu.europa.ec.eudi:pid:1\",\n                            \"type\": \"string\"\n                        },\n                        \"path\": [\n                            \"$.vct\"\n                        ]\n                    }\n                ]\n            },\n            \"format\": {\n                \"vc+sd-jwt\": {\n                    \"kb-jwt_alg_values\": [\n                        \"ES256\",\n                        \"ES256K\",\n                        \"EdDSA\"\n                    ],\n                    \"sd-jwt_alg_values\": [\n                        \"ES256\",\n                        \"ES256K\",\n                        \"EdDSA\"\n                    ]\n                }\n            },\n            \"id\": \"pid_credential\"\n        }\
		]\n}","fieldName":"presentation_definition"}}}}`),
			Headers:        map[string]string{"Content-Type": "application/json", "Authorization": authToken},
			Delay:          0,
			Timeout:        5 * time.Second,
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"Invalid JSON body",
			},
			NotExpectedContent: []string{"start"},
			TestAppFactory:     setupTestApp,
		},
		{
			Name:           "OpenID4VP Test - Invalid Token",
			Method:         "POST",
			URL:            "/api/compliance/check/OpenID4VP_Wallet/OpenID_Foundation/save-variables-and-start",
			Body:           strings.NewReader(`{"sd_jwt_vc:did:request_uri_signed:direct_post.json":{"format":"variables","data":{"oid_description":{"type":"string","value":"jikj","fieldName":"description"},"oid_alias":{"type":"string","value":"knnkn","fieldName":"testalias"},"oid_client_id":{"type":"string","value":"did:web:app.altme.io:issuer","fieldName":"client_id"},"oid_client_jwks":{"type":"object","value":"{\n    \"keys\": [\n        {\n            \"kty\": \"EC\",\n            \"alg\": \"ES256\",\n            \"crv\": \"P-256\",\n            \"d\": \"GSbo9TpmGaLgxxO6RNx6QnvcfykQJS7vUVgTe8vy9W0\",\n            \"x\": \"m5uKsE35t3sP7gjmirUewufx2Gt2n6J7fSW68apB2Lo\",\n            \"y\": \"-V54TpMI8RbpB40hbAocIjnaHX5WP6NHjWkHfdCSAyU\"\n        }\n    ]\n}","fieldName":"jwks"},"oid_client_presentation_definition":{"type":"object","value":"{\n    \"id\": \"two_sd_jwt\",\n    \"input_descriptors\": [\n        {\n            \"constraints\": {\n                \"fields\": [\n                    {\n                        \"filter\": {\n                            \"const\": \"urn:eu.europa.ec.eudi:pid:1\",\n                            \"type\": \"string\"\n                        },\n                        \"path\": [\n                            \"$.vct\"\n                        ]\n                    }\n                ]\n            },\n            \"format\": {\n                \"vc+sd-jwt\": {\n                    \"kb-jwt_alg_values\": [\n                        \"ES256\",\n                        \"ES256K\",\n                        \"EdDSA\"\n                    ],\n                    \"sd-jwt_alg_values\": [\n                        \"ES256\",\n                        \"ES256K\",\n                        \"EdDSA\"\n                    ]\n                }\n            },\n            \"id\": \"pid_credential\"\n        }\
		]\n}","fieldName":"presentation_definition"}}}}`),
			Headers:        map[string]string{"Content-Type": "application/json", "Authorization	": "invalid_token"},
			Delay:          0,
			Timeout:        5 * time.Second,
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"The request requires valid record authorization token.",
			},
			NotExpectedContent: []string{"start"},
			TestAppFactory:     setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}

}
