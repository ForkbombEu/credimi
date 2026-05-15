// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/openapi-go/openapi3"
)

type TestBodyRequest struct {
	Name string `json:"name"`
}

type TestBodyResponse struct {
	ID string `json:"id"`
}

func TestBuildOpenAPISpec_ParametersAndResponses(t *testing.T) {
	route := RouteInfo{
		Method:      http.MethodGet,
		Path:        "/api/things/{thingId}/logs",
		OperationID: "thing.logs",
		AuthHeaders: []authHeaderParam{
			{
				Name:        "Authorization",
				Required:    false,
				Description: "Bearer token for authentication.",
			},
			{
				Name:        "Credimi-Api-Key",
				Required:    false,
				Description: "API key.",
			},
		},
		QuerySearchAttributes: []routing.QuerySearchAttribute{
			{Name: "action", Required: true, Description: "action"},
		},
		OutputSchema: TestBodyResponse{},
	}
	route.PathParams = extractPathParams(route.Path)
	route.HasInputBody = false

	spec, err := buildOpenAPISpec([]RouteInfo{route})
	require.NoError(t, err)

	op := requireOperation(t, spec, route.Path, route.Method)

	pathParam := findParam(op.Parameters, "thingId", openapi3.ParameterIn("path"))
	require.NotNil(t, pathParam)
	require.NotNil(t, pathParam.Required)
	assert.True(t, *pathParam.Required)

	queryParam := findParam(op.Parameters, "action", openapi3.ParameterIn("query"))
	require.NotNil(t, queryParam)
	require.NotNil(t, queryParam.Required)
	assert.True(t, *queryParam.Required)

	authParam := findParam(op.Parameters, "Authorization", openapi3.ParameterIn("header"))
	require.NotNil(t, authParam)
	apiKeyParam := findParam(op.Parameters, "Credimi-Api-Key", openapi3.ParameterIn("header"))
	require.NotNil(t, apiKeyParam)
	if apiKeyParam.Required != nil {
		assert.False(t, *apiKeyParam.Required)
	}
	require.NotNil(t, op.ID)
	assert.Equal(t, "thing.logs", *op.ID)

	_, ok := op.Responses.MapOfResponseOrRefValues["200"]
	require.True(t, ok)
	require.NotNil(t, op.Responses.Default)
}

func TestBuildOpenAPISpec_RequestBodyRequired(t *testing.T) {
	route := RouteInfo{
		Method:       http.MethodPost,
		Path:         "/api/things",
		OperationID:  "things.create",
		InputSchema:  TestBodyRequest{},
		OutputSchema: TestBodyResponse{},
		HasInputBody: true,
	}

	spec, err := buildOpenAPISpec([]RouteInfo{route})
	require.NoError(t, err)

	op := requireOperation(t, spec, route.Path, route.Method)
	require.NotNil(t, op.RequestBody)
	require.NotNil(t, op.RequestBody.RequestBody)
	require.NotNil(t, op.RequestBody.RequestBody.Required)
	assert.True(t, *op.RequestBody.RequestBody.Required)

	mt, ok := op.RequestBody.RequestBody.Content["application/json"]
	require.True(t, ok)
	require.NotNil(t, mt.Schema)
	require.NotNil(t, op.ID)
	assert.Equal(t, "things.create", *op.ID)

	authParam := findParam(op.Parameters, "Authorization", openapi3.ParameterIn("header"))
	require.Nil(t, authParam)
	require.NotNil(t, op.Responses.Default)
}

func requireOperation(t *testing.T, spec *openapi3.Spec, path, method string) openapi3.Operation {
	t.Helper()

	pathItem, ok := spec.Paths.MapOfPathItemValues[path]
	require.True(t, ok, "missing path %s", path)

	op, ok := pathItem.MapOfOperationValues[strings.ToLower(method)]
	require.True(t, ok, "missing %s operation for %s", method, path)

	return op
}

func findParam(
	params []openapi3.ParameterOrRef,
	name string,
	in openapi3.ParameterIn,
) *openapi3.Parameter {
	for _, param := range params {
		if param.Parameter == nil {
			continue
		}
		if param.Parameter.Name == name && param.Parameter.In == in {
			return param.Parameter
		}
	}
	return nil
}

func TestNeedsAuth(t *testing.T) {
	require.False(t, needsAuth(false, nil))
	require.False(t, needsAuth(true, []string{apis.DefaultRequireAuthMiddlewareId}))
	require.True(t, needsAuth(true, []string{"other"}))
}

func TestAuthHeadersForRoute(t *testing.T) {
	userHeaders := authHeadersForRoute(true, nil, nil, nil)
	require.Len(t, userHeaders, 2)
	require.Equal(t, "Authorization", userHeaders[0].Name)
	require.False(t, userHeaders[0].Required)
	require.Equal(t, "Credimi-Api-Key", userHeaders[1].Name)
	require.False(t, userHeaders[1].Required)

	require.Empty(t, authHeadersForRoute(
		true,
		nil,
		nil,
		[]string{apis.DefaultRequireAuthMiddlewareId},
	))

	internalHeaders := authHeadersForRoute(
		false,
		nil,
		[]*hook.Handler[*core.RequestEvent]{middlewares.RequireInternalAdminAPIKey()},
		nil,
	)
	require.Len(t, internalHeaders, 1)
	require.Equal(t, "Credimi-Api-Key", internalHeaders[0].Name)
	require.True(t, internalHeaders[0].Required)

	eitherHeaders := authHeadersForRoute(
		false,
		nil,
		[]*hook.Handler[*core.RequestEvent]{middlewares.RequireInternalAdminOrAuth()},
		nil,
	)
	require.Len(t, eitherHeaders, 2)
	require.Equal(t, "Authorization", eitherHeaders[0].Name)
	require.Equal(t, "Credimi-Api-Key", eitherHeaders[1].Name)
}

func TestExportNameAndParamFieldName(t *testing.T) {
	require.Equal(t, "Param", exportName(""))
	require.Equal(t, "Param", exportName("!!!"))
	require.Equal(t, "FooBarBaz", exportName("foo_bar-baz"))
	require.Equal(t, "1abc", exportName("1abc"))
	require.Equal(t, "QueryFoo", paramFieldName("Query", "foo"))
}

func TestUniqueFieldName(t *testing.T) {
	used := map[string]struct{}{}
	require.Equal(t, "Field", uniqueFieldName("Field", used))
	require.Equal(t, "Field2", uniqueFieldName("Field", used))
	used["Field3"] = struct{}{}
	require.Equal(t, "Field4", uniqueFieldName("Field", used))
}

func TestBuildTagAndSanitize(t *testing.T) {
	tag := buildTag("json", "a`b\"c", true, "line1\nline2")
	require.Contains(t, string(tag), `json:"ab'c"`)
	require.Contains(t, string(tag), `required:"true"`)
	require.Contains(t, string(tag), `description:"line1 line2"`)
}

func TestPathHelpers(t *testing.T) {
	require.True(t, isPathParam("{id}"))
	require.False(t, isPathParam("id"))
	require.Equal(t, []string{"id", "name"}, extractPathParams("/things/{id}/sub/{name}"))
	require.Equal(t, "/", joinOpenAPIPath("/"))
	require.Equal(t, "/api/v1/items", joinOpenAPIPath("/api/", "/v1/", "/items/"))
}
