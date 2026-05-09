// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

func ParseEntityIDs(yamlStr string) (workflowengine.EntityIDs, error) {
    entityIDs := workflowengine.EntityIDs{
        Actions:           []string{},
        Versions:          []string{},
        Credentials:       []string{},
        UseCases:          []string{},
        ConformanceChecks: []string{},
        CustomChecks:      []string{},
    }

    if strings.TrimSpace(yamlStr) == "" {
        return entityIDs, nil
    }

    wfDef, err := ParseWorkflow(yamlStr)
    if err != nil {
        return entityIDs, err
    }

    actionsSet := make(map[string]struct{})
    versionsSet := make(map[string]struct{})
    credentialsSet := make(map[string]struct{})
    useCasesSet := make(map[string]struct{})
    conformanceChecksSet := make(map[string]struct{})
    customChecksSet := make(map[string]struct{})

    collectEntityIDs := func(step StepSpec) {
        if step.With.Payload == nil {
            return
        }

        // Actions
        if rawActionID, ok := step.With.Payload["action_id"]; ok {
            if s, ok := rawActionID.(string); ok && s != "" {
                actionsSet[canonify.NormalizePath(s)] = struct{}{}
            }
        }

        // Versions
        if rawVersionID, ok := step.With.Payload["version_id"]; ok {
            if s, ok := rawVersionID.(string); ok && s != "" {
                versionsSet[canonify.NormalizePath(s)] = struct{}{}
            }
        }

        // Credentials
        if rawCredentialID, ok := step.With.Payload["credential_id"]; ok {
            if s, ok := rawCredentialID.(string); ok && s != "" {
                credentialsSet[canonify.NormalizePath(s)] = struct{}{}
            }
        }

        // Use Cases
        if rawUseCaseID, ok := step.With.Payload["use_case_id"]; ok {
            if s, ok := rawUseCaseID.(string); ok && s != "" {
                useCasesSet[canonify.NormalizePath(s)] = struct{}{}
            }
        }

        if rawCheckID, ok := step.With.Payload["check_id"]; ok {
            if s, ok := rawCheckID.(string); ok && s != "" {
                normalized := canonify.NormalizePath(s)
                switch step.Use {
                case "conformance-check":
                    conformanceChecksSet[normalized] = struct{}{}
                case "custom-check":
                    customChecksSet[normalized] = struct{}{}
                }
            }
        }
    }

    for _, step := range wfDef.Steps {
        collectEntityIDs(step.StepSpec)
        for _, onErr := range step.OnError {
            collectEntityIDs(onErr.StepSpec)
        }
        for _, onSuccess := range step.OnSuccess {
            collectEntityIDs(onSuccess.StepSpec)
        }
    }

    entityIDs.Actions = setToSlice(actionsSet)
    entityIDs.Versions = setToSlice(versionsSet)
    entityIDs.Credentials = setToSlice(credentialsSet)
    entityIDs.UseCases = setToSlice(useCasesSet)
    entityIDs.ConformanceChecks = setToSlice(conformanceChecksSet)
    entityIDs.CustomChecks = setToSlice(customChecksSet)

    return entityIDs, nil
}

func setToSlice(set map[string]struct{}) []string {
    if len(set) == 0 {
        return []string{}
    }
    result := make([]string, 0, len(set))
    for k := range set {
        result = append(result, k)
    }
    sort.Strings(result)
    return result
}
