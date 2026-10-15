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

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/workflow"
	"gopkg.in/yaml.v3"
)

const mobileBeforeActionsPayloadKey = "before_actions"

// beforeActionFlowKeys are the flow-header keys a before action may carry. Its
// commands run inside the step's flow, which keeps its own header, so any other
// key (onFlowStart, onFlowComplete, ...) would silently stop applying.
var beforeActionFlowKeys = map[string]bool{"appId": true, "name": true, "tags": true, "env": true}

// mobileBeforeAction is a stored wallet action resolved for composition.
type mobileBeforeAction struct {
	ActionID   string
	Code       string
	Parameters map[string]any
}

// composeStepBeforeActions turns a mobile-automation step's `before_actions`
// into the leading commands of its own flow, so they run in the same Maestro
// session as the step's action, before it. Each before action runs in a
// `runFlow` whose env maps its parameters, which the step carries as
// `before_<n>_<name>` parameters so that they resolve like the step's own.
// Codes are fetched once per action id and run.
func composeStepBeforeActions(
	ctx workflow.Context,
	step *pipeline.StepSpec,
	ownerNamespace string,
	codes map[string]string,
) error {
	raw, ok := step.With.Payload[mobileBeforeActionsPayloadKey]
	if !ok {
		return nil
	}
	entries, err := workflowengine.DecodeOutput[[]workflows.MobileAutomationBeforeAction](raw)
	if err != nil {
		return invalidBeforeActionsError(step.ID, err.Error())
	}
	before := make([]mobileBeforeAction, 0, len(entries))
	for i, entry := range entries {
		actionID := strings.TrimSpace(entry.ActionID)
		if actionID == "" {
			return invalidBeforeActionsError(
				step.ID,
				fmt.Sprintf("before_actions[%d] has no action_id", i),
			)
		}
		code, cached := codes[actionID]
		if !cached {
			action, err := fetchMobileAction(ctx, ownerNamespace, actionID)
			if err != nil {
				return err
			}
			code = workflowengine.AsString(action["code"])
			if code == "" {
				errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
				return workflowengine.NewAppError(workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf(
						"%s: wallet action %s has no code for before action of step %s",
						errCode.Description,
						actionID,
						step.ID,
					),
				})
			}
			codes[actionID] = code
		}
		parameters := make(map[string]any, len(entry.Parameters))
		for name, value := range entry.Parameters {
			parameters[name] = value
		}
		before = append(before, mobileBeforeAction{
			ActionID:   actionID,
			Code:       code,
			Parameters: parameters,
		})
	}

	mainCode := workflowengine.AsString(step.With.Payload["action_code"])
	parameters := stepParameters(step.With.Payload["parameters"])
	composed, err := composeMobileFlow(mainCode, parameters, before)
	if err != nil {
		return invalidBeforeActionsError(step.ID, err.Error())
	}
	SetPayloadValue(&step.With.Payload, "action_code", composed)
	if len(parameters) > 0 {
		SetPayloadValue(&step.With.Payload, "parameters", parameters)
	}
	delete(step.With.Payload, mobileBeforeActionsPayloadKey)
	return nil
}

func invalidBeforeActionsError(stepID, reason string) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf("invalid before_actions for step %s: %s", stepID, reason),
	})
}

// composeMobileFlow prepends each before action to the main Maestro flow as a
// `runFlow` block and adds the before actions' parameters to `parameters`
// (mutated in place) under `before_<n>_<name>`. A before action's screenshots
// are dropped: the step's evidence is its own action's screenshots only, which
// a precondition's screens would otherwise pad.
func composeMobileFlow(
	mainCode string,
	parameters map[string]any,
	before []mobileBeforeAction,
) (string, error) {
	header, commands, err := splitMaestroFlow(mainCode)
	if err != nil {
		return "", fmt.Errorf("step action: %w", err)
	}
	mainAppID := mappingValue(header, "appId")

	blocks := make([]*yaml.Node, 0, len(before)+len(commands.Content))
	for i, action := range before {
		beforeHeader, beforeCommands, err := splitMaestroFlow(action.Code)
		if err != nil {
			return "", fmt.Errorf("before action %s: %w", action.ActionID, err)
		}
		env, err := beforeActionEnv(beforeHeader, mainAppID)
		if err != nil {
			return "", fmt.Errorf("before action %s: %w", action.ActionID, err)
		}
		for _, name := range slices.Sorted(maps.Keys(action.Parameters)) {
			key := fmt.Sprintf("before_%d_%s", i, name)
			if _, exists := parameters[key]; exists {
				return "", fmt.Errorf("parameter %s is already set on the step", key)
			}
			parameters[key] = action.Parameters[name]
			setMappingValue(env, name, scalarNode("${"+key+"}"))
		}
		dropScreenshots(beforeCommands)

		runFlow := &yaml.Node{Kind: yaml.MappingNode}
		if len(env.Content) > 0 {
			runFlow.Content = append(runFlow.Content, scalarNode("env"), env)
		}
		runFlow.Content = append(runFlow.Content, scalarNode("commands"), beforeCommands)
		blocks = append(blocks, &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{scalarNode("runFlow"), runFlow},
		})
	}
	commands.Content = append(blocks, commands.Content...)

	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if header != nil {
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

// beforeActionEnv checks a before action's header and returns a copy of its
// env defaults, which become the env of its runFlow block.
func beforeActionEnv(header *yaml.Node, mainAppID string) (*yaml.Node, error) {
	env := &yaml.Node{Kind: yaml.MappingNode}
	if header == nil {
		return env, nil
	}
	for i := 0; i+1 < len(header.Content); i += 2 {
		key, value := header.Content[i].Value, header.Content[i+1]
		if !beforeActionFlowKeys[key] {
			return nil, fmt.Errorf("flow key %q cannot apply inside another flow", key)
		}
		switch key {
		case "appId":
			if value.Value != mainAppID {
				return nil, fmt.Errorf(
					"it drives app %q, the step's action drives %q",
					value.Value,
					mainAppID,
				)
			}
		case "env":
			if value.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("its env is not a mapping")
			}
			env.Content = append(env.Content, value.Content...)
		}
	}
	return env, nil
}

// dropScreenshots removes every takeScreenshot command, at any depth.
func dropScreenshots(node *yaml.Node) {
	if node.Kind == yaml.SequenceNode {
		kept := node.Content[:0]
		for _, item := range node.Content {
			if item.Kind == yaml.MappingNode && len(item.Content) > 0 &&
				item.Content[0].Value == "takeScreenshot" {
				continue
			}
			kept = append(kept, item)
		}
		node.Content = kept
	}
	for _, child := range node.Content {
		dropScreenshots(child)
	}
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

func setMappingValue(node *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1] = value
			return
		}
	}
	node.Content = append(node.Content, scalarNode(key), value)
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}
