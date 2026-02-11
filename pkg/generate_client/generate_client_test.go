// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/routing"
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
		Method:        http.MethodGet,
		GoHandlerName: "handlers.HandleTestQueryRoute",
		Path:          "/api/things/{thingId}/logs",
		QuerySearchAttributes: []routing.QuerySearchAttribute{
			{Name: "action", Required: true, Description: "action"},
		},
		AuthenticationRequired: true,
		OutputSchema:           TestBodyResponse{},
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

	_, ok := op.Responses.MapOfResponseOrRefValues["200"]
	require.True(t, ok)
	require.NotNil(t, op.Responses.Default)
}

func TestBuildOpenAPISpec_RequestBodyRequired(t *testing.T) {
	route := RouteInfo{
		Method:                 http.MethodPost,
		GoHandlerName:          "handlers.HandleTestBodyRoute",
		Path:                   "/api/things",
		InputSchema:            TestBodyRequest{},
		OutputSchema:           TestBodyResponse{},
		HasInputBody:           true,
		AuthenticationRequired: false,
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
