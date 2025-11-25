// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	outputPath string
)

// NewSchemaCmd creates the "schema" subcommand for pipeline
func NewSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate YAML Schema for the pipeline",
		Long:  "Generates a YAML Schema with oneOf validation for each step type based on the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			schema, err := generatePipelineSchema()
			if err != nil {
				return fmt.Errorf("failed to generate schema: %w", err)
			}

			jsonData, err := json.MarshalIndent(schema, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal schema to JSON: %w", err)
			}

			var output []byte
			if outputPath != "" &&
				(len(outputPath) > 5 && outputPath[len(outputPath)-5:] == ".yaml" ||
					len(outputPath) > 4 && outputPath[len(outputPath)-4:] == ".yml") {
				var jsonMap map[string]any
				if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
					return fmt.Errorf("failed to unmarshal JSON: %w", err)
				}
				output, err = yaml.Marshal(jsonMap)
				if err != nil {
					return fmt.Errorf("failed to marshal schema to YAML: %w", err)
				}
			} else {
				output = jsonData
			}
			if outputPath != "" {
				err = os.WriteFile(outputPath, output, 0644)
				if err != nil {
					return fmt.Errorf("failed to write schema to file: %w", err)
				}
				fmt.Printf("✅ Pipeline schema generated successfully and saved to %s\n", outputPath)
			} else {
				fmt.Println(string(output))
			}

			return nil
		},
	}

	cmd.Flags().
		StringVarP(&outputPath, "output", "o", "", "Output file path (if not specified, prints to stdout)")

	return cmd
}

func generatePipelineSchema() (map[string]any, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := reflector.Reflect(&pipeline.WorkflowDefinition{})
	stepDefSchema := reflector.Reflect(&pipeline.StepDefinition{})
	activityOptionsSchema := stepDefSchema.Properties.Value("activity_options")
	activityOptionsJSON, _ := json.Marshal(activityOptionsSchema)
	var activityOptionsMap map[string]any
	json.Unmarshal(activityOptionsJSON, &activityOptionsMap)

	var oneOfSchemas []map[string]any
	for _, stepKey := range sortedRegistryKeys() {
		stepSchema := generateSingleStepSchema(&reflector, stepKey)
		if stepSchema != nil {
			oneOfSchemas = append(oneOfSchemas, stepSchema)
		}
	}

	if stepsProperty, ok := schema.Properties.Get("steps"); ok && stepsProperty != nil {
		if stepsProperty.Items != nil {
			newItemsSchema := &jsonschema.Schema{
				OneOf: make([]*jsonschema.Schema, len(oneOfSchemas)),
			}
			for i, variant := range oneOfSchemas {
				variantBytes, _ := json.Marshal(variant)
				var variantSchema jsonschema.Schema
				json.Unmarshal(variantBytes, &variantSchema)
				newItemsSchema.OneOf[i] = &variantSchema
			}
			// Replace the entire items schema
			stepsProperty.Items = newItemsSchema
		}
	}

	if customChecksProperty, ok := schema.Properties.Get("custom_checks"); ok &&
		customChecksProperty != nil {
		if customChecksProperty.AdditionalProperties != nil {
			wbSchema := customChecksProperty.AdditionalProperties
			if stepsProperty, ok := wbSchema.Properties.Get("steps"); ok && stepsProperty != nil {
				if stepsProperty.Items != nil {
					newItemsSchema := &jsonschema.Schema{
						OneOf: make([]*jsonschema.Schema, len(oneOfSchemas)),
					}
					for i, variant := range oneOfSchemas {
						variantBytes, _ := json.Marshal(variant)
						var variantSchema jsonschema.Schema
						json.Unmarshal(variantBytes, &variantSchema)
						newItemsSchema.OneOf[i] = &variantSchema
					}
					stepsProperty.Items = newItemsSchema
				}
			}
		}
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		return nil, err
	}
	schemaMap["$defs"] = map[string]any{
		"ActivityOptions": activityOptionsMap,
	}

	return schemaMap, nil
}

func generateSingleStepSchema(reflector *jsonschema.Reflector, stepKey string) map[string]any {
	factory, ok := registry.Registry[stepKey]
	if !ok {
		return nil
	}

	payloadType := factory.PayloadType
	if factory.PipelinePayloadType != nil {
		payloadType = factory.PipelinePayloadType
	}

	payloadSchema := reflector.Reflect(reflect.New(payloadType).Interface())

	stepVariant := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{"type": "string"},
			"use": map[string]any{
				"type":  "string",
				"const": stepKey,
			},
			"with": map[string]any{
				"type": "object",
			},
			"activity_options": map[string]any{
				"$ref": "#/$defs/ActivityOptions",
			},
			"metadata": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
			"continue_on_error": map[string]any{
				"type": "boolean",
			},
		},
		"required":             []string{"id", "use", "with"},
		"additionalProperties": false,
	}

	// --- Merge payload properties & required fields ---
	payloadBytes, _ := json.Marshal(payloadSchema)
	var payloadMap map[string]any
	_ = json.Unmarshal(payloadBytes, &payloadMap)

	with := stepVariant["properties"].(map[string]any)["with"].(map[string]any)

	props := map[string]any{
		"config": map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		},
	}

	if p, ok := payloadMap["properties"].(map[string]any); ok {
		for k, v := range p {
			props[k] = v
		}
	}

	with["properties"] = props

	// Preserve required fields from payload
	if req, ok := payloadMap["required"].([]any); ok && len(req) > 0 {
		with["required"] = req
	}
	if stepKey == "mobile-automation" {
		with := []map[string]any{
			{
				"type": "object",
				"properties": map[string]any{
					"action_id": map[string]any{"type": "string"},
					"video":     map[string]any{"type": "boolean"},
					"parameters": map[string]any{
						"type":                 "object",
						"additionalProperties": map[string]any{"type": "string"},
					},
					"config": map[string]any{
						"type":                 "object",
						"additionalProperties": true,
					},
				},
				"required":             []string{"action_id"},
				"additionalProperties": false,
			},
			{
				"type": "object",
				"properties": map[string]any{
					"action_id":  map[string]any{"type": "string"},
					"version_id": map[string]any{"type": "string"},
					"video":      map[string]any{"type": "boolean"},
					"parameters": map[string]any{
						"type":                 "object",
						"additionalProperties": map[string]any{"type": "string"},
					},
					"config": map[string]any{
						"type":                 "object",
						"additionalProperties": true,
					},
				},
				"required":             []string{"action_id", "version_id"},
				"additionalProperties": false,
			},
			{
				"type": "object",
				"properties": map[string]any{
					"version_id":  map[string]any{"type": "string"},
					"action_code": map[string]any{"type": "string"},
					"video":       map[string]any{"type": "boolean"},
					"parameters": map[string]any{
						"type":                 "object",
						"additionalProperties": map[string]any{"type": "string"},
					},
					"config": map[string]any{
						"type":                 "object",
						"additionalProperties": true,
					},
				},
				"required":             []string{"action_code", "version_id"},
				"additionalProperties": false,
			},
		}

		stepVariant["properties"].(map[string]any)["with"] = map[string]any{
			"oneOf": with,
		}
	}

	return stepVariant
}

func sortedRegistryKeys() []string {
	keys := make([]string, 0, len(registry.Registry))
	for key := range registry.Registry {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
