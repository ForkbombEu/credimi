// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	api "github.com/forkbombeu/credimi/pkg/internal/apis"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/invopop/jsonschema"
	"gopkg.in/yaml.v3"
)

// =================================================================
// =============== DATA STRUCTURES
// =================================================================

type RouteInfo struct {
	FuncName               string
	Method                 string
	GoHandlerName          string
	Path                   string
	InputType              string
	OutputType             string
	QuerySearchAttributes  []routing.QuerySearchAttribute
	PathParams             []string
	HasInputBody           bool
	Summary                string
	Description            string
	Tags                   []string
	AuthenticationRequired bool
}

type (
	OpenAPI struct {
		OpenAPI    string               `json:"openapi"           yaml:"openapi"`
		Info       Info                 `json:"info"              yaml:"info"`
		Servers    []Server             `json:"servers,omitempty" yaml:"servers,omitempty"`
		Paths      map[string]*PathItem `json:"paths"             yaml:"paths"`
		Components Components           `json:"components"        yaml:"components"`
	}
	Server struct {
		URL         string `json:"url"                   yaml:"url"`
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
	}
	Info struct {
		Title       string  `json:"title"             yaml:"title"`
		Version     string  `json:"version"           yaml:"version"`
		Description string  `json:"description"       yaml:"description"`
		Contact     Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	}
	Contact struct {
		Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
		Email string `json:"email,omitempty" yaml:"email,omitempty"`
		URL   string `json:"url,omitempty"   yaml:"url,omitempty"`
	}
	PathItem struct {
		Get    *Operation `json:"get,omitempty"    yaml:"get,omitempty"`
		Post   *Operation `json:"post,omitempty"   yaml:"post,omitempty"`
		Put    *Operation `json:"put,omitempty"    yaml:"put,omitempty"`
		Patch  *Operation `json:"patch,omitempty"  yaml:"patch,omitempty"`
		Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	}
	Operation struct {
		Tags        []string            `json:"tags,omitempty"        yaml:"tags,omitempty"`
		Summary     string              `json:"summary"               yaml:"summary"`
		Description string              `json:"description,omitempty" yaml:"description,omitempty"`
		OperationID string              `json:"operationId"           yaml:"operationId"`
		Parameters  []Parameter         `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
		RequestBody *RequestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
		Responses   map[string]Response `json:"responses"             yaml:"responses"`
	}
	Parameter struct {
		Name        string             `json:"name"                  yaml:"name"`
		In          string             `json:"in"                    yaml:"in"`
		Description string             `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool               `json:"required"              yaml:"required"`
		Schema      *jsonschema.Schema `json:"schema"                yaml:"schema"`
	}
	RequestBody struct {
		Description string             `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool               `json:"required"              yaml:"required"`
		Content     map[string]Content `json:"content"               yaml:"content"`
	}
	Content struct {
		Schema Ref `json:"schema" yaml:"schema"`
	}
	Response struct {
		Description string             `json:"description"       yaml:"description"`
		Content     map[string]Content `json:"content,omitempty" yaml:"content,omitempty"`
	}
	Ref struct {
		Ref string `json:"$ref" yaml:"$ref"`
	}
	Components struct {
		Schemas map[string]*jsonschema.Schema `json:"schemas" yaml:"schemas"`
	}
)

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

	typesToProcess := make(map[string]interface{})

	typesToProcess["APIError"] = apierror.APIError{}

	log.Println("Processing routes...")
	for _, group := range routeGroups {
		for _, route := range group.Routes {
			// Use url.JoinPath for proper URL path joining instead of file path.Join
			joinedPath := joinOpenAPIPath(group.BaseURL, route.Path)
			r := RouteInfo{
				Method:                 route.Method,
				Path:                   joinedPath,
				GoHandlerName:          getFuncName(route.Handler),
				Summary:                route.Summary,
				Description:            route.Description,
				QuerySearchAttributes:  route.QuerySearchAttributes,
				AuthenticationRequired: group.AuthenticationRequired,
				// Tags:          route.Tags,
			}

			if route.RequestSchema != nil {
				typeName := reflect.TypeOf(route.RequestSchema).Name()
				typesToProcess[typeName] = route.RequestSchema
				r.InputType = typeName
			}

			if route.ResponseSchema != nil {
				typeName := reflect.TypeOf(route.ResponseSchema).Name()
				typesToProcess[typeName] = route.ResponseSchema
				r.OutputType = typeName
			} else {
				r.OutputType = "any"
			}

			r.FuncName = handlerToFuncName(r.GoHandlerName)
			r.PathParams = extractPathParams(r.Path)
			r.HasInputBody = r.InputType != "" &&
				(r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH")
			routes = append(routes, r)
		}
	}

	log.Println("Generating OpenAPI YAML documentation...")
	generateOpenAPIYAML(routes, typesToProcess)

	log.Println("✅ Generation complete.")
}

// =================================================================
// =============== GENERATOR FUNCTIONS
// =================================================================

func generateOpenAPIYAML(routes []RouteInfo, typesToProcess map[string]interface{}) {
	reflector := new(jsonschema.Reflector)
	reflector.RequiredFromJSONSchemaTags = true
	reflector.ExpandedStruct = true
	schemas := make(map[string]*jsonschema.Schema)
	for name, typ := range typesToProcess {
		schema := reflector.Reflect(typ)
		if schema.Definitions != nil {
			for defName, defSchema := range schema.Definitions {
				schemas[defName] = defSchema
			}
			schema.Definitions = nil
		}
		schemas[name] = schema
	}

	paths := make(map[string]*PathItem)
	openapiPathRegex := regexp.MustCompile(`{([^{}]+)}`)
	for _, route := range routes {
		openapiPath := openapiPathRegex.ReplaceAllString(route.Path, "{$1}")
		if !strings.HasPrefix(openapiPath, "/") {
			openapiPath = "/" + openapiPath
		}
		if _, ok := paths[openapiPath]; !ok {
			paths[openapiPath] = &PathItem{}
		}
		operation := buildOperation(route)
		assignOperationToPath(paths[openapiPath], route.Method, operation)
		if route.AuthenticationRequired {
			addAuthHeader(paths[openapiPath])
		}
	}

	doc := OpenAPI{
		OpenAPI: "3.0.1",
		Info: Info{
			Title:       "credimi 👀 API Gateway",
			Version:     "1.4.0",
			Description: `credimi API Gateway for managing EUDI-ARF compliance checks...`,
			Contact: Contact{
				Name:  "credimi Support",
				Email: "support@forkbomb.eu",
				URL:   "https://forkbomb.solutions",
			},
		},
		Servers: []Server{
			{URL: "https://credimi.io", Description: "Production server"},
			{URL: "https://demo.credimi.io", Description: "Demo server"},
			{URL: "http://localhost:8090/", Description: "Localhost server"},
		},
		Paths: paths,
		Components: Components{
			Schemas: schemas,
		},
	}

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		log.Fatalf("FATAL: Failed to marshal intermediate JSON: %v", err)
	}

	var genericData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &genericData); err != nil {
		log.Fatalf("FATAL: Failed to unmarshal intermediate JSON: %v", err)
	}

	cleanRefs(genericData)

	outputPath := "../docs/public/API/openapi.yml"
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(genericData); err != nil {
		log.Fatalf("FATAL: Failed to marshal final YAML: %v", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		log.Fatalf("FATAL: Failed to write OpenAPI file '%s': %v", outputPath, err)
	}
	log.Printf("✅ OpenAPI YAML documentation successfully generated at: %s", outputPath)
}

func buildOperation(route RouteInfo) *Operation {
	operation := &Operation{
		Summary:     route.Summary,
		Description: route.Description,
		OperationID: handlerToFuncName(route.GoHandlerName),
		Tags:        route.Tags,
		Responses:   make(map[string]Response),
	}
	for _, pName := range route.PathParams {
		operation.Parameters = append(operation.Parameters, Parameter{
			Name:        pName,
			In:          "path",
			Required:    true,
			Description: fmt.Sprintf("The ID for the %s.", pName),
			Schema:      &jsonschema.Schema{Type: "string"},
		})
	}
	if route.InputType != "" {
		if route.HasInputBody {
			operation.RequestBody = &RequestBody{
				Required: true,
				Content: map[string]Content{
					"application/json": {
						Schema: Ref{Ref: "#/components/schemas/" + route.InputType},
					},
				},
			}
		} else if route.QuerySearchAttributes != nil {
			for _, attr := range route.QuerySearchAttributes {
				operation.Parameters = append(operation.Parameters, Parameter{
					Name:        attr.Name,
					In:          "query",
					Required:    attr.Required,
					Description: attr.Description,
					Schema:      &jsonschema.Schema{Type: "string"},
				})
			}
		}
	} else {
		if route.QuerySearchAttributes != nil {
			for _, attr := range route.QuerySearchAttributes {
				operation.Parameters = append(operation.Parameters, Parameter{
					Name:        attr.Name,
					In:          "query",
					Required:    attr.Required,
					Description: attr.Description,
					Schema:      &jsonschema.Schema{Type: "string"},
				})
			}
		}
	}
	if route.OutputType != "" && route.OutputType != "any" {
		operation.Responses["200"] = Response{
			Description: "Successful response",
			Content: map[string]Content{
				"application/json": {Schema: Ref{Ref: "#/components/schemas/" + route.OutputType}},
			},
		}
	} else {
		operation.Responses["200"] = Response{Description: "Successful response without a body"}
	}
	operation.Responses["default"] = Response{
		Description: "An unexpected error occurred.",
		Content: map[string]Content{
			"application/json": {Schema: Ref{Ref: "#/components/schemas/APIError"}},
		},
	}
	return operation
}

func assignOperationToPath(pathItem *PathItem, method string, op *Operation) {
	switch strings.ToUpper(method) {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "PATCH":
		pathItem.Patch = op
	case "DELETE":
		pathItem.Delete = op
	}
}

func addAuthHeader(pathItem *PathItem) {
	authParam := Parameter{
		Name:        "Authorization",
		In:          "header",
		Description: "Bearer token for authentication",
		Required:    true,
		Schema:      &jsonschema.Schema{Type: "string"},
	}
	if pathItem.Get != nil {
		pathItem.Get.Parameters = append(pathItem.Get.Parameters, authParam)
	}
	if pathItem.Post != nil {
		pathItem.Post.Parameters = append(pathItem.Post.Parameters, authParam)
	}
	if pathItem.Put != nil {
		pathItem.Put.Parameters = append(pathItem.Put.Parameters, authParam)
	}
	if pathItem.Patch != nil {
		pathItem.Patch.Parameters = append(pathItem.Patch.Parameters, authParam)
	}
	if pathItem.Delete != nil {
		pathItem.Delete.Parameters = append(pathItem.Delete.Parameters, authParam)
	}
}

func cleanRefs(v interface{}) {
	switch vv := v.(type) {
	case map[string]interface{}:
		for k, val := range vv {
			switch k {
			case "$defs":
				delete(vv, k)
			case "$ref":
				refStr, ok := val.(string)
				if ok && strings.HasPrefix(refStr, "#/$defs/") {
					vv[k] = "#/components/schemas/" + strings.TrimPrefix(refStr, "#/$defs/")
				}
			default:
				cleanRefs(val)
			}
		}
	case []interface{}:
		for _, item := range vv {
			cleanRefs(item)
		}
	}
}

// =================================================================
// =============== HELPER FUNCTIONS
// =================================================================

func getFuncName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
func handlerToFuncName(fullHandlerName string) string {
	parts := strings.Split(fullHandlerName, ".")
	name := parts[len(parts)-1]
	name = strings.TrimPrefix(name, "Handle")
	if name == "" {
		return "unnamedRoute"
	}
	return strings.ToLower(name[:1]) + name[1:]
}
func extractPathParams(path string) []string {
	var params []string
	re := regexp.MustCompile(`{([^{}]+)}`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

func joinOpenAPIPath(base string, parts ...string) string {
	seg := []string{strings.TrimRight(base, "/")}
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
