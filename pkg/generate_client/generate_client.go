// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	api "github.com/forkbombeu/credimi/pkg/internal/apis"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
)

// =================================================================
// =============== DATA STRUCTURES
// =================================================================

type RouteInfo struct {
	Method                 string
	Path                   string
	InputSchema            any
	OutputSchema           any
	QuerySearchAttributes  []routing.QuerySearchAttribute
	PathParams             []string
	HasInputBody           bool
	Summary                string
	Description            string
	Tags                   []string
	AuthenticationRequired bool
}

// =================================================================
// =============== MAIN LOGIC
// =================================================================

func main() {
	log.Println("Starting API generation...")

	routeGroups := api.RouteGroups
	if len(routeGroups) == 0 {
		log.Fatal(
			"FATAL: No route groups found. Did you register them in api.AllRouteGroups using an init() function?",
		)
	}

	totalRoutes := 0
	for _, group := range routeGroups {
		totalRoutes += len(group.Routes)
	}
	routes := make([]RouteInfo, 0, totalRoutes)

	log.Println("Processing routes...")
	for _, group := range routeGroups {
		for _, route := range group.Routes {
			// Use url.JoinPath for proper URL path joining instead of file path.Join
			joinedPath := joinOpenAPIPath(group.BaseURL, route.Path)
			r := RouteInfo{
				Method:                route.Method,
				Path:                  joinedPath,
				Summary:               route.Summary,
				Description:           route.Description,
				QuerySearchAttributes: route.QuerySearchAttributes,
				AuthenticationRequired: needsAuth(
					group.AuthenticationRequired,
					route.ExcludedMiddlewares,
				),
				// Tags:          route.Tags,
			}

			if route.RequestSchema != nil {
				r.InputSchema = route.RequestSchema
			}

			if route.ResponseSchema != nil {
				r.OutputSchema = route.ResponseSchema
			}

			r.PathParams = extractPathParams(r.Path)
			r.HasInputBody = r.InputSchema != nil &&
				(r.Method == http.MethodPost ||
					r.Method == http.MethodPut ||
					r.Method == http.MethodPatch)
			routes = append(routes, r)
		}
	}

	log.Println("Generating OpenAPI YAML documentation...")
	generateOpenAPIYAML(routes)

	log.Println("✅ Generation complete.")
}

// =================================================================
// =============== GENERATOR FUNCTIONS
// =================================================================

func generateOpenAPIYAML(routes []RouteInfo) {
	spec, err := buildOpenAPISpec(routes)
	if err != nil {
		log.Fatalf("FATAL: Failed to build OpenAPI spec: %v", err)
	}
	yamlBytes, err := spec.MarshalYAML()
	if err != nil {
		log.Fatalf("FATAL: Failed to marshal OpenAPI YAML: %v", err)
	}
	outputPath := "../docs/public/API/openapi.yml"
	if err := os.WriteFile(outputPath, yamlBytes, 0644); err != nil {
		log.Fatalf("FATAL: Failed to write OpenAPI file '%s': %v", outputPath, err)
	}
	log.Printf("✅ OpenAPI YAML documentation successfully generated at: %s", outputPath)
}

func buildOpenAPISpec(routes []RouteInfo) (*openapi3.Spec, error) {
	spec := &openapi3.Spec{
		Openapi: "3.0.3",
		Info: openapi3.Info{
			Title:   "credimi 👀 API Gateway",
			Version: "1.4.0",
			Description: stringPtr(
				`credimi API Gateway for managing EUDI-ARF compliance checks...`,
			),
			Contact: &openapi3.Contact{
				Name:  stringPtr("credimi Support"),
				Email: stringPtr("support@forkbomb.eu"),
				URL:   stringPtr("https://forkbomb.solutions"),
			},
		},
		Servers: []openapi3.Server{
			buildServer("https://credimi.io", "Production server"),
			buildServer("https://demo.credimi.io", "Demo server"),
			buildServer("http://localhost:8090/", "Localhost server"),
		},
		Paths: openapi3.Paths{},
	}

	reflector := openapi3.Reflector{Spec: spec}
	for _, route := range routes {
		operationContext, err := reflector.NewOperationContext(route.Method, route.Path)
		if err != nil {
			return nil, fmt.Errorf(
				"create operation context for %s %s: %w",
				route.Method,
				route.Path,
				err,
			)
		}
		setOperationMetadata(operationContext, route)
		if err := addOperationRequest(operationContext, route); err != nil {
			return nil, err
		}
		addOperationResponses(operationContext, route)
		if err := reflector.AddOperation(operationContext); err != nil {
			return nil, fmt.Errorf("add operation for %s %s: %w", route.Method, route.Path, err)
		}
	}

	return spec, nil
}

func setOperationMetadata(operation openapi.OperationInfo, route RouteInfo) {
	if route.Summary != "" {
		operation.SetSummary(route.Summary)
	}
	if route.Description != "" {
		operation.SetDescription(route.Description)
	}
	operation.SetID(pathToOperationID(route.Path))
	if len(route.Tags) > 0 {
		operation.SetTags(route.Tags...)
	}
}

func addOperationRequest(operation openapi.OperationContext, route RouteInfo) error {
	req := buildRequestStructure(route)

	if req == nil {
		if !route.HasInputBody {
			return nil
		}
	} else {
		operation.AddReqStructure(req)
	}

	if route.HasInputBody {
		if route.InputSchema == nil {
			return fmt.Errorf("request schema missing for %s %s", route.Method, route.Path)
		}
		operation.AddReqStructure(
			route.InputSchema,
			openapi.WithContentType("application/json"),
			withRequiredRequestBody(true),
		)
	}

	return nil
}

func addOperationResponses(operation openapi.OperationContext, route RouteInfo) {
	if route.OutputSchema != nil {
		operation.AddRespStructure(
			route.OutputSchema,
			openapi.WithHTTPStatus(http.StatusOK),
			openapi.WithContentType("application/json"),
			withResponseDescription("Successful response"),
		)
	} else {
		operation.AddRespStructure(
			nil,
			openapi.WithHTTPStatus(http.StatusOK),
			withResponseDescription("Successful response without a body"),
		)
	}
	operation.AddRespStructure(
		apierror.APIError{},
		openapi.WithContentType("application/json"),
		withDefaultResponseDescription("An unexpected error occurred."),
	)
}

func buildServer(url, description string) openapi3.Server {
	return openapi3.Server{
		URL:         url,
		Description: stringPtr(description),
	}
}

func buildRequestStructure(route RouteInfo) any {
	fields := make(
		[]reflect.StructField,
		0,
		len(route.PathParams)+len(route.QuerySearchAttributes)+2,
	)
	used := map[string]struct{}{}

	for _, param := range route.PathParams {
		fieldName := uniqueFieldName(paramFieldName("Path", param), used)
		fields = append(fields, reflect.StructField{
			Name: fieldName,
			Type: reflect.TypeOf(""),
			Tag:  buildTag("path", param, true, fmt.Sprintf("The ID for the %s.", param)),
		})
	}

	if route.AuthenticationRequired {
		fieldName := uniqueFieldName("HeaderAuthorization", used)
		fields = append(fields, reflect.StructField{
			Name: fieldName,
			Type: reflect.TypeOf(""),
			Tag:  buildTag("header", "Authorization", true, "Bearer token for authentication"),
		})
	}

	if !route.HasInputBody {
		for _, attr := range route.QuerySearchAttributes {
			fieldName := uniqueFieldName(paramFieldName("Query", attr.Name), used)
			fields = append(fields, reflect.StructField{
				Name: fieldName,
				Type: reflect.TypeOf(""),
				Tag:  buildTag("query", attr.Name, attr.Required, attr.Description),
			})
		}
	}

	if len(fields) == 0 {
		return nil
	}

	reqType := reflect.StructOf(fields)
	return reflect.New(reqType).Interface()
}

func withRequiredRequestBody(required bool) openapi.ContentOption {
	return openapi.WithCustomize(func(cor openapi.ContentOrReference) {
		reqBody, ok := cor.(*openapi3.RequestBodyOrRef)
		if !ok || reqBody.RequestBody == nil {
			return
		}
		reqBody.RequestBody.WithRequired(required)
	})
}

func withResponseDescription(description string) openapi.ContentOption {
	return func(cu *openapi.ContentUnit) {
		cu.Description = description
	}
}

func withDefaultResponseDescription(description string) openapi.ContentOption {
	return func(cu *openapi.ContentUnit) {
		cu.IsDefault = true
		cu.Description = description
	}
}

// =================================================================
// =============== HELPER FUNCTIONS
// =================================================================

func stringPtr(val string) *string {
	return &val
}

func needsAuth(groupAuth bool, excluded []string) bool {
	if !groupAuth {
		return false
	}
	for _, middlewareID := range excluded {
		if middlewareID == apis.DefaultRequireAuthMiddlewareId {
			return false
		}
	}
	return true
}

func paramFieldName(prefix, name string) string {
	return prefix + exportName(name)
}

func exportName(name string) string {
	if name == "" {
		return "Param"
	}
	var b strings.Builder
	capNext := true
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if capNext {
				b.WriteString(strings.ToUpper(string(r)))
				capNext = false
			} else {
				b.WriteRune(r)
			}
			continue
		}
		capNext = true
	}
	if b.Len() == 0 {
		return "Param"
	}
	return b.String()
}

func uniqueFieldName(name string, used map[string]struct{}) string {
	if _, exists := used[name]; !exists {
		used[name] = struct{}{}
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", name, i)
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func buildTag(kind, name string, required bool, description string) reflect.StructTag {
	parts := []string{fmt.Sprintf(`%s:"%s"`, kind, sanitizeTagValue(name))}
	if required {
		parts = append(parts, `required:"true"`)
	}
	if description != "" {
		parts = append(parts, fmt.Sprintf(`description:"%s"`, sanitizeTagValue(description)))
	}
	return reflect.StructTag(strings.Join(parts, " "))
}

func sanitizeTagValue(val string) string {
	val = strings.ReplaceAll(val, "`", "")
	val = strings.ReplaceAll(val, `"`, `'`)
	val = strings.ReplaceAll(val, "\n", " ")
	return val
}

func pathToOperationID(path string) string {
	segments := splitPathSegments(path)
	if len(segments) == 0 {
		return "route.execute"
	}

	last := segments[len(segments)-1]
	if isPathParam(last) {
		objectWords := pathResourceWords(segments, false)
		if len(objectWords) == 0 {
			return "route.get"
		}
		return fmt.Sprintf("%s.get", lowerCamelFromWords(objectWords))
	}

	if len(segments) == 1 {
		objectWords := segmentWords(last)
		if len(objectWords) == 0 {
			return "route.list"
		}
		return fmt.Sprintf("%s.list", lowerCamelFromWords(objectWords))
	}

	if isPathParam(segments[len(segments)-2]) &&
		countPathParams(segments) == 1 &&
		isLikelyCollectionSegment(last) {
		objectWords := pathResourceWords(segments, true)
		if len(objectWords) == 0 {
			return "route.list"
		}
		return fmt.Sprintf("%s.list", lowerCamelFromWords(objectWords))
	}

	action := strings.Join(segmentWords(last), "")
	if action == "" {
		action = "execute"
	}
	objectWords := pathResourceWords(segments[:len(segments)-1], false)
	if len(objectWords) == 0 {
		objectWords = []string{"route"}
	}

	return fmt.Sprintf("%s.%s", lowerCamelFromWords(objectWords), action)
}

func pathResourceWords(segments []string, keepLeafPlural bool) []string {
	literals := make([][]string, 0, len(segments))
	for _, segment := range segments {
		if isPathParam(segment) {
			continue
		}
		words := segmentWords(segment)
		if len(words) == 0 {
			continue
		}
		literals = append(literals, words)
	}

	if len(literals) == 0 {
		return nil
	}

	out := make([]string, 0, len(literals)*2)
	last := len(literals) - 1
	for i, words := range literals {
		if i != last || !keepLeafPlural {
			words = singularizeWords(words)
		}
		out = append(out, words...)
	}

	return out
}

func splitPathSegments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if len(segments) == 0 && part == "api" {
			continue
		}
		if len(segments) == 0 && part == "my" {
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

func isPathParam(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func countPathParams(segments []string) int {
	count := 0
	for _, segment := range segments {
		if isPathParam(segment) {
			count++
		}
	}
	return count
}

func isLikelyCollectionSegment(segment string) bool {
	words := segmentWords(segment)
	if len(words) == 0 {
		return false
	}
	last := words[len(words)-1]
	return singularizeWord(last) != last
}

func segmentWords(segment string) []string {
	segment = strings.ToLower(strings.TrimSpace(segment))
	if segment == "" {
		return nil
	}

	var b strings.Builder
	for _, r := range segment {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}

	words := strings.Fields(b.String())
	if len(words) == 1 && strings.HasPrefix(words[0], "api") && len(words[0]) > 3 {
		return []string{"api", words[0][3:]}
	}
	return words
}

func singularizeWords(words []string) []string {
	if len(words) == 0 {
		return words
	}
	out := append([]string(nil), words...)
	out[len(out)-1] = singularizeWord(out[len(out)-1])
	return out
}

func singularizeWord(word string) string {
	switch {
	case strings.HasSuffix(word, "ies") && len(word) > 3:
		return word[:len(word)-3] + "y"
	case strings.HasSuffix(word, "sses"),
		strings.HasSuffix(word, "ches"),
		strings.HasSuffix(word, "shes"),
		strings.HasSuffix(word, "xes"),
		strings.HasSuffix(word, "zes"):
		return word[:len(word)-2]
	case strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") && len(word) > 1:
		return word[:len(word)-1]
	default:
		return word
	}
}

func lowerCamelFromWords(words []string) string {
	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(words[0])
	for _, word := range words[1:] {
		if word == "" {
			continue
		}
		b.WriteString(strings.ToUpper(word[:1]))
		if len(word) > 1 {
			b.WriteString(word[1:])
		}
	}

	return b.String()
}

func extractPathParams(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	params := make([]string, 0, len(parts))
	for _, part := range parts {
		if isPathParam(part) {
			params = append(params, strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}"))
		}
	}
	return params
}

func joinOpenAPIPath(base string, parts ...string) string {
	seg := make([]string, 0, 1+len(parts))
	seg = append(seg, strings.TrimRight(base, "/"))

	for _, p := range parts {
		seg = append(seg, strings.Trim(p, "/"))
	}
	out := strings.Join(seg, "/")
	if !strings.HasPrefix(out, "/") {
		out = "/" + out
	}
	// remove trailing slash unless it's just "/"
	if len(out) > 1 {
		out = strings.TrimRight(out, "/")
	}
	return out
}
