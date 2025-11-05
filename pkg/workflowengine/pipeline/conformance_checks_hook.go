// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
	"gopkg.in/yaml.v3"
)

func ConformanceCheckHook(
	ctx workflow.Context,
	steps *[]StepDefinition,
	input workflowengine.WorkflowInput,
) error {
	logger := workflow.GetLogger(ctx)

	for i := range *steps {
		step := &(*steps)[i]

		if step.Use != "conformance-check" {
			continue
		}
		logger.Info("ConformanceCheckHook: processing step", "step", step.ID)

		checkID, ok := step.With.Payload["check_id"].Value.(string)
		if !ok {
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				fmt.Sprintf("missing or invalid check_id for step %s", step.ID),
			)
		}

		rootDir := utils.GetEnvironmentVariable("ROOT_DIR", true)
		configTemplatePath := fmt.Sprintf("%s/config_templates/%s.yaml", rootDir, checkID)
		content, err := os.ReadFile(configTemplatePath)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.ReadFileFailed]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("failed to read template file %s: %v", configTemplatePath, err),
			)
		}

		extractedContent, err := extractCredimiJSON(string(content))
		if err != nil {
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.TemplateRenderFailed],
				fmt.Sprintf("failed to extract credimi JSON from %s: %v", configTemplatePath, err),
			)
		}

		var tpl map[string]any
		if err := yaml.Unmarshal([]byte(extractedContent), &tpl); err != nil {
			errCode := errorcodes.Codes[errorcodes.TemplateRenderFailed]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("failed to parse template YAML %s: %v", configTemplatePath, err),
			)
		}

		tpl = extractValues(tpl).(map[string]any)
		parts := strings.Split(filepath.ToSlash(checkID), "/")
		var suite, standard, checkName string
		if len(parts) >= 2 {
			suite = parts[len(parts)-2]
			standard = parts[0]
			checkName = parts[len(parts)-1]
		}
		memo := map[string]any{
			"author":   suite,
			"standard": standard,
			"test":     checkName,
		}
		SetConfigValue(&step.With.Config, "memo", memo)

		userMail, ok := input.Config["user_mail"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
				"missing or invalid user_mail in workflow config",
			)
		}

		defaultPayload := map[string]any{
			"user_mail": userMail,
			"suite":     suite,
		}

		var suiteTemplatePath string
		switch suite {
		case "openid_conformance_suite":
			variant := map[string]any{}
			if v, ok := tpl["variant"].(map[string]any); ok {
				for _, key := range []string{"credential_format", "client_id_prefix", "request_method", "response_mode"} {
					if val, ok := v[key]; ok {
						variant[key] = val
					}
				}
			}
			variantJSON, err := json.Marshal(variant)
			if err != nil {
				errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("failed to marshal variant JSON for step %s: %v", step.ID, err),
				)
			}

			form := map[string]any{}
			if f, ok := tpl["form"].(map[string]any); ok {
				for k, v := range f {
					form[k] = v
				}
			}

			var testVal string
			if tVal, ok := tpl["test"].(string); ok {
				testVal = tVal
			}

			defaultPayload["variant"] = string(variantJSON)
			defaultPayload["form"] = form
			defaultPayload["test"] = testVal

			suiteTemplatePath = workflows.OpenIDNetStepCITemplatePathv1_0

		case "ewc":

			var sessionID string
			if sID, ok := tpl["session_id"].(string); ok {
				sessionID = sID
			}

			defaultPayload["session_id"] = sessionID
			suiteTemplatePath = workflows.EWCTemplateFolderPath + "/" + checkName + ".yaml"
		case "eudiw":

			var id string
			if tID, ok := tpl["id"].(string); ok {
				id = tID
			}
			var nonce string
			if tNonce, ok := tpl["nonce"].(string); ok {
				nonce = tNonce
			}

			defaultPayload["id"] = id
			defaultPayload["nonce"] = nonce
			suiteTemplatePath = workflows.EudiwTemplateFolderPath + "/" + checkName + ".yaml"
		default:
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("missing or invalid suite for step %s", step.ID),
			)
		}

		templatePath := fmt.Sprintf("%s/%s", rootDir, suiteTemplatePath)
		template, err := os.ReadFile(templatePath)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.ReadFileFailed]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("failed to read template file %s: %v", templatePath, err),
			)
		}
		MergePayload(step.With.Payload, defaultPayload)
		SetConfigValue(&step.With.Config, "template", string(template))
	}

	return nil
}

func extractCredimiJSON(yamlContent string) (string, error) {
	re := regexp.MustCompile(`\{\{\s*credimi\s*` + "`" + `([\s\S]*?)` + "`" + `\s*([a-zA-Z0-9_]+)?\s*\}\}`)
	var firstErr error

	extracted := re.ReplaceAllStringFunc(yamlContent, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		jsonPart := strings.TrimSpace(sub[1])
		var functionName string
		if len(sub) >= 3 {
			functionName = strings.TrimSpace(sub[2])
		}

		var obj map[string]any
		if err := json.Unmarshal([]byte(jsonPart), &obj); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("invalid credimi JSON: %w", err)
			}
			return match
		}

		if functionName != "" {
			obj["field_function"] = functionName
		}

		result, _ := json.Marshal(obj)
		return string(result)
	})

	if re.MatchString(extracted) {
		return extractCredimiJSON(extracted)
	}
	return extracted, firstErr
}

func extractValues(node any) any {
	switch n := node.(type) {
	case map[string]any:

		if fieldType, ok := n["field_type"].(string); ok {
			defaultVal := n["field_default_value"]

			// Check if there's a function to apply
			if functionName, ok := n["field_function"].(string); ok {
				switch functionName {
				case "uuidv4":
					return uuid.New().String()
				default:
				}
			}
			switch fieldType {
			case "string":
				if s, ok := defaultVal.(string); ok {
					return s
				}
				return ""
			case "object":
				if s, ok := defaultVal.(string); ok && strings.TrimSpace(s) != "" {
					var obj any
					if err := json.Unmarshal([]byte(s), &obj); err == nil {
						return obj
					}
				} else if obj, ok := defaultVal.(map[string]any); ok {
					return obj
				}
				return map[string]any{}
			default:
				if defaultVal != nil {
					return defaultVal
				}
				return ""
			}
		}
		res := make(map[string]any)
		for k, v := range n {
			res[k] = extractValues(v)
		}
		return res

	case []any:
		for i, v := range n {
			n[i] = extractValues(v)
		}
		return n

	case string:
		s := strings.TrimSpace(n)
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			var obj map[string]any
			if err := json.Unmarshal([]byte(s), &obj); err == nil {
				if _, hasFieldType := obj["field_type"]; hasFieldType {
					return extractValues(obj)
				}
				return obj
			}
		}
		return n

	default:
		return n
	}
}
