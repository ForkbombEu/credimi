// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
)

// Extracts outputs from all previous steps
func ExtractPipelineOutput(dataCtx map[string]any) map[string]any {
	result := make(map[string]any)
	
	for key, value := range dataCtx {
		if key == "inputs" {
			continue
		}
		
		if stepOutput, ok := value.(map[string]any); ok {
			if outputs, exists := stepOutput["outputs"]; exists {
				result[key] = outputs
			} else {
				result[key] = value
			}
		} else {
			result[key] = value
		}
	}
	
	return result
}

func EnrichActivityPayload(payload any, use string, pipelineOutput map[string]any) (any, error) {
    switch use {
    case "http-request":
        httpPayload, ok := payload.(*activities.HTTPActivityPayload)
        if !ok {
            return payload, nil
        }
        enriched := *httpPayload
        enriched.Body = pipelineOutput
        return &enriched, nil
        
    case "email":
        emailPayload, ok := payload.(*activities.SendMailActivityPayload)
        if !ok {
            return payload, nil
        }
        enriched := *emailPayload
        bodyJSON, err := json.Marshal(pipelineOutput)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal pipeline output: %w", err)
        }
        enriched.Body = string(bodyJSON)
        return &enriched, nil
        
    default:
        return payload, nil
    }
}
