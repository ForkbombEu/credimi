// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/workflow"
	"gopkg.in/yaml.v3"
)

const mobileActionsPayloadKey = "actions"

// actionFlowKeys are the flow-header keys an action of `actions` may carry.
// Every action runs as a `runFlow` block of one composed flow, whose header is
// only the shared appId, so any other key (onFlowStart, onFlowComplete, ...)
// would silently stop applying.
var actionFlowKeys = map[string]bool{"appId": true, "name": true, "tags": true, "env": true}

// mobileAction is one entry of a step's `actions`, with its code resolved.
type mobileAction struct {
	Label      string
	Code       string
	Parameters map[string]any
}

// composeStepActions turns a mobile-automation step's `actions` into one
// Maestro flow, stored as the step's action_code, so the actions run in order
// in a single Maestro session. Each action runs in a `runFlow` block. An
// action's own parameters are passed as step parameters
// `action_<n>_<name>`, so that they resolve like the step's own, and mapped
// back to `<name>` in the block's env: they take precedence over the step's
// parameters, which every action sees, and those over the defaults in the
// action's own `env`. Stored actions are fetched once per id and run.
func composeStepActions(
	ctx workflow.Context,
	step *pipeline.StepSpec,
	ownerNamespace string,
	records map[string]map[string]any,
) error {
	raw, ok := step.With.Payload[mobileActionsPayloadKey]
	if !ok {
		return nil
	}
	for _, single := range []string{"action_id", "action_code"} {
		if _, set := step.With.Payload[single]; set {
			return invalidActionsError(step.ID, single+" cannot be combined with actions")
		}
	}
	entries, err := workflowengine.DecodeOutput[[]workflows.MobileAutomationAction](raw)
	if err != nil {
		return invalidActionsError(step.ID, err.Error())
	}
	if len(entries) == 0 {
		return invalidActionsError(step.ID, "actions is empty")
	}

	actions := make([]mobileAction, 0, len(entries))
	allStored := true
	for i, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		code := entry.Code
		parameters := make(map[string]any, len(entry.Parameters))
		for name, value := range entry.Parameters {
			parameters[name] = value
		}
		switch {
		case id != "" && strings.TrimSpace(code) != "":
			return invalidActionsError(step.ID, fmt.Sprintf("actions[%d] has both id and code", i))
		case strings.TrimSpace(code) != "":
			allStored = false
			actions = append(actions, mobileAction{
				Label:      fmt.Sprintf("actions[%d]", i),
				Code:       code,
				Parameters: parameters,
			})
			continue
		case id == "":
			return invalidActionsError(
				step.ID,
				fmt.Sprintf("actions[%d] has neither id nor code", i),
			)
		}

		id = canonify.NormalizePath(id)
		record, cached := records[id]
		if !cached {
			record, err = fetchMobileAction(ctx, ownerNamespace, id)
			if err != nil {
				return err
			}
			records[id] = record
		}
		storedCode := workflowengine.AsString(record["code"])
		if storedCode == "" {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			return workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf(
					"%s: wallet action %s has no code for step %s",
					errCode.Description,
					id,
					step.ID,
				),
			})
		}
		markExternalInstall(step, record)
		actions = append(actions, mobileAction{
			Label:      id,
			Code:       storedCode,
			Parameters: parameters,
		})
	}

	parameters := stepParameters(step.With.Payload["parameters"])
	composed, err := composeMobileFlow(parameters, actions)
	if err != nil {
		return invalidActionsError(step.ID, err.Error())
	}
	SetPayloadValue(&step.With.Payload, "action_code", composed)
	if allStored {
		SetPayloadValue(&step.With.Payload, "stored_action_code", true)
	}
	if len(parameters) > 0 {
		SetPayloadValue(&step.With.Payload, "parameters", parameters)
	}
	delete(step.With.Payload, mobileActionsPayloadKey)
	return nil
}

func invalidActionsError(stepID, reason string) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf("invalid actions for step %s: %s", stepID, reason),
	})
}

// composeMobileFlow writes the actions as the `runFlow` blocks of one flow
// and adds their own parameters to `parameters` (mutated in place) under
// `action_<n>_<name>`. Every action keeps its screenshots.
func composeMobileFlow(parameters map[string]any, actions []mobileAction) (string, error) {
	appID := ""
	commands := &yaml.Node{Kind: yaml.SequenceNode}
	for i, action := range actions {
		header, actionCommands, err := splitMaestroFlow(action.Code)
		if err != nil {
			return "", fmt.Errorf("action %s: %w", action.Label, err)
		}
		actionAppID := mappingValue(header, "appId")
		if i == 0 {
			appID = actionAppID
		} else if actionAppID != appID {
			return "", fmt.Errorf(
				"action %s drives app %q, action %s drives %q",
				action.Label,
				actionAppID,
				actions[0].Label,
				appID,
			)
		}
		defaults, err := actionEnvDefaults(header)
		if err != nil {
			return "", fmt.Errorf("action %s: %w", action.Label, err)
		}

		env := &yaml.Node{Kind: yaml.MappingNode}
		for j := 0; j+1 < len(defaults.Content); j += 2 {
			name := defaults.Content[j].Value
			_, ownValue := action.Parameters[name]
			_, stepValue := parameters[name]
			if ownValue || stepValue {
				continue
			}
			env.Content = append(env.Content, defaults.Content[j], defaults.Content[j+1])
		}
		for _, name := range slices.Sorted(maps.Keys(action.Parameters)) {
			key := fmt.Sprintf("action_%d_%s", i, name)
			if _, exists := parameters[key]; exists {
				return "", fmt.Errorf("parameter %s is already set on the step", key)
			}
			parameters[key] = action.Parameters[name]
			env.Content = append(env.Content, scalarNode(name), scalarNode("${"+key+"}"))
		}

		runFlow := &yaml.Node{Kind: yaml.MappingNode}
		if len(env.Content) > 0 {
			runFlow.Content = append(runFlow.Content, scalarNode("env"), env)
		}
		runFlow.Content = append(runFlow.Content, scalarNode("commands"), actionCommands)
		commands.Content = append(commands.Content, &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{scalarNode("runFlow"), runFlow},
		})
	}

	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if appID != "" {
		header := &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{scalarNode("appId"), scalarNode(appID)},
		}
		if err := encoder.Encode(header); err != nil {
			return "", fmt.Errorf("encode flow header: %w", err)
		}
	}
	if err := encoder.Encode(commands); err != nil {
		return "", fmt.Errorf("encode flow commands: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("encode flow: %w", err)
	}
	return out.String(), nil
}

// stepParameters copies a step's parameters, which YAML decodes as
// map[string]any and Go callers may set as map[string]string.
func stepParameters(raw any) map[string]any {
	parameters := map[string]any{}
	switch values := raw.(type) {
	case map[string]any:
		maps.Copy(parameters, values)
	case map[string]string:
		for name, value := range values {
			parameters[name] = value
		}
	}
	return parameters
}

// splitMaestroFlow returns a flow's header (nil when it has none) and its
// command list.
func splitMaestroFlow(code string) (*yaml.Node, *yaml.Node, error) {
	decoder := yaml.NewDecoder(strings.NewReader(code))
	var documents []*yaml.Node
	for {
		var document yaml.Node
		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("parse Maestro flow: %w", err)
		}
		if len(document.Content) == 0 {
			continue
		}
		documents = append(documents, document.Content[0])
	}
	var header, commands *yaml.Node
	switch len(documents) {
	case 1:
		commands = documents[0]
	case 2:
		header, commands = documents[0], documents[1]
		if header.Kind != yaml.MappingNode {
			return nil, nil, fmt.Errorf("the Maestro flow header is not a mapping")
		}
	default:
		return nil, nil, fmt.Errorf(
			"a Maestro flow has a header and a command list, found %d documents",
			len(documents),
		)
	}
	if commands.Kind != yaml.SequenceNode {
		return nil, nil, fmt.Errorf("the Maestro flow has no command list")
	}
	return header, commands, nil
}

// actionEnvDefaults checks an action's header and returns its env defaults.
func actionEnvDefaults(header *yaml.Node) (*yaml.Node, error) {
	env := &yaml.Node{Kind: yaml.MappingNode}
	if header == nil {
		return env, nil
	}
	for i := 0; i+1 < len(header.Content); i += 2 {
		key, value := header.Content[i].Value, header.Content[i+1]
		if !actionFlowKeys[key] {
			return nil, fmt.Errorf("flow key %q cannot apply inside another flow", key)
		}
		if key != "env" {
			continue
		}
		if value.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("its env is not a mapping")
		}
		env.Content = append(env.Content, value.Content...)
	}
	return env, nil
}

func mappingValue(node *yaml.Node, key string) string {
	if node == nil {
		return ""
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1].Value
		}
	}
	return ""
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}
