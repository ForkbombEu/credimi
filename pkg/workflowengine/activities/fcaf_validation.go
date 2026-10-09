// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipelinehistory"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/activity"
)

const (
	FCAFValidationActivityName       = "Run FCAF validation"
	DefaultFCAFValidationCatalogRoot = "config_templates/fcaf/wallet_solution/relying_party"
)

type FCAFValidationActivityInput struct {
	TestID      string         `json:"test_id,omitempty"          yaml:"test_id,omitempty"`
	TestIDs     []string       `json:"test_ids,omitempty"         yaml:"test_ids,omitempty"`
	Suite       string         `json:"suite,omitempty"            yaml:"suite,omitempty"`
	CatalogRoot string         `json:"catalog_root,omitempty"     yaml:"catalog_root,omitempty"`
	Pipeline    map[string]any `json:"pipeline_outputs,omitempty" yaml:"pipeline_outputs,omitempty"`
	Runtime     map[string]any `json:"runtime,omitempty"          yaml:"runtime,omitempty"`
}

// FCAFValidationActivityOutput is the compact report Temporal records. The full report,
// with evidence values, is stored on the pipeline result as the fcaf_report file.
type FCAFValidationActivityOutput struct {
	// Report is the public report with every evidence value removed.
	Report engine.Report `json:"report"`
	// ReportSHA256 is the sha256 of the stored fcaf_report file.
	ReportSHA256  string                  `json:"report_sha256"`
	EvidenceIndex []FCAFEvidenceReference `json:"evidence_index,omitempty"`
}

// FCAFEvidenceReference points from one pipeline_outputs leaf to the history event that
// recorded the step output it was resolved from.
type FCAFEvidenceReference struct {
	// Source is the pipeline_outputs key, e.g. "pipeline.pid.presentation.sdjwt.all-claims".
	Source string `json:"source"`
	// Path is the dotted path of the leaf inside Source, e.g. "output.pid_sdjwt".
	Path string `json:"path"`
	// Ref is the expression body, e.g. "obtain.outputs.body | optional".
	Ref string `json:"ref"`
	// StepID is the first segment of the expression's initial value.
	StepID string `json:"step_id"`
	// EventID is the history event that recorded StepID's output; 0 when none did.
	EventID int64 `json:"event_id"`
	// SHA256 is the hex sha256 of the JSON of the resolved leaf; empty when it is nil.
	SHA256 string `json:"sha256"`
}

type FCAFValidationActivity struct {
	workflowengine.BaseActivity
	app           core.App
	catalogLoader func(root string) (*catalog.Catalog, error)
	outputKind    pipelinehistory.OutputKindFunc
}

func NewFCAFValidationActivity(
	app core.App,
	outputKind pipelinehistory.OutputKindFunc,
) *FCAFValidationActivity {
	return &FCAFValidationActivity{
		BaseActivity:  workflowengine.BaseActivity{Name: FCAFValidationActivityName},
		app:           app,
		catalogLoader: catalog.Load,
		outputKind:    outputKind,
	}
}

func (a *FCAFValidationActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute resolves pipeline_outputs against the step outputs recorded in the run's own
// history, runs the FCAF engine, stores the full report on the pipeline result and returns
// a compact report that references the evidence instead of copying it.
func (a *FCAFValidationActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFValidationActivityInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	testIDs := normalizeValidationTestIDs(payload)
	if len(testIDs) == 0 {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(
			fmt.Errorf("test_id or test_ids is required"),
		)
	}
	if len(payload.Pipeline) == 0 {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(
			fmt.Errorf("pipeline_outputs is required"),
		)
	}
	leaves := collectFCAFEvidenceLeaves(payload.Pipeline)
	dataCtx := map[string]any{}
	eventIDs := map[string]int64{}
	if len(leaves) > 0 {
		run, err := a.loadRun(ctx)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
			return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf("load pipeline history: %v", err),
			})
		}
		dataCtx = run.DataContext()
		eventIDs = run.EventIDs
	}
	resolvedValue, err := pipelineinternal.ResolveExpressions(payload.Pipeline, dataCtx)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.PipelineInputError]
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: fmt.Sprintf("resolve pipeline_outputs: %v", err),
		})
	}
	resolved, _ := resolvedValue.(map[string]any)

	catalogRoot := payload.CatalogRoot
	if catalogRoot == "" {
		catalogRoot = resolveDefaultFCAFValidationCatalogRoot()
	}
	suite := payload.Suite
	if suite == "" {
		suite = "wallet_solution/relying_party"
	}
	loader := a.catalogLoader
	if loader == nil {
		loader = catalog.Load
	}
	cat, err := loader(catalogRoot)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
			Message: err.Error(),
		})
	}
	fcafEngine, err := engine.New(nil)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
			Message: fmt.Sprintf("create fcaf engine: %v", err),
		})
	}
	bundle := evidence.Bundle{PipelineOutputs: resolved, Runtime: payload.Runtime}
	report, err := fcafEngine.ExecuteCatalog(
		ctx,
		cat,
		testIDs,
		suite,
		payload.Runtime,
		bundle,
	)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
			Message: fmt.Sprintf("execute fcaf engine: %v", err),
		})
	}
	report.PopulateDerivedViews()
	full := report.PublicReport()

	reportSHA256, err := a.storeReport(ctx, input.Config, full)
	if err != nil {
		return workflowengine.ActivityResult{}, fcafReportStoreError(&a.BaseActivity, err)
	}

	return workflowengine.ActivityResult{
		Output: FCAFValidationActivityOutput{
			Report:        compactFCAFReport(full),
			ReportSHA256:  reportSHA256,
			EvidenceIndex: buildFCAFEvidenceIndex(leaves, resolved, eventIDs),
		},
	}, nil
}

func (a *FCAFValidationActivity) loadRun(ctx context.Context) (*pipelinehistory.Run, error) {
	info := activity.GetInfo(ctx)
	c, err := temporalclient.GetTemporalClientWithNamespace(info.Namespace)
	if err != nil {
		return nil, err
	}
	return pipelinehistory.Load(
		ctx,
		c,
		info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID,
		a.outputKind,
	)
}

// storeReport stores the full report on the top-level run's pipeline result and returns
// the sha256 of the stored file. A child pipeline stores it on its root run, which owns the
// pipeline_results row.
func (a *FCAFValidationActivity) storeReport(
	ctx context.Context,
	config map[string]string,
	report engine.Report,
) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("%w: %w", errFCAFReportEncode, err)
	}
	workflowID := config[workflowengine.TelemetryRootWorkflowIDKey]
	runID := config[workflowengine.TelemetryRootRunIDKey]
	if workflowID == "" || runID == "" {
		info := activity.GetInfo(ctx)
		workflowID = info.WorkflowExecution.ID
		runID = info.WorkflowExecution.RunID
	}
	return storeFCAFReport(ctx, a.app, workflowID, runID, data)
}

// compactFCAFReport returns report with every evidence value removed.
func compactFCAFReport(report engine.Report) engine.Report {
	if report.Evidence == nil {
		return report
	}
	compact := make(engine.EvidenceMap, len(report.Evidence))
	for name, record := range report.Evidence {
		record.Value = nil
		compact[name] = record
	}
	report.Evidence = compact
	return report
}

// fcafEvidenceLeaf is a pipeline_outputs string that contains expressions.
type fcafEvidenceLeaf struct {
	source string
	path   []any // map keys (string) and slice indexes (int) below source
	refs   []string
}

func collectFCAFEvidenceLeaves(pipelineOutputs map[string]any) []fcafEvidenceLeaf {
	var leaves []fcafEvidenceLeaf
	var walk func(source string, path []any, value any)
	walk = func(source string, path []any, value any) {
		switch typed := value.(type) {
		case string:
			if refs := pipelineinternal.ExpressionRefs(typed); len(refs) > 0 {
				leaves = append(leaves, fcafEvidenceLeaf{
					source: source,
					path:   slices.Clone(path),
					refs:   refs,
				})
			}
		case map[string]any:
			for _, key := range slices.Sorted(maps.Keys(typed)) {
				walk(source, append(path, key), typed[key])
			}
		case []any:
			for index, item := range typed {
				walk(source, append(path, index), item)
			}
		}
	}
	for _, source := range slices.Sorted(maps.Keys(pipelineOutputs)) {
		walk(source, nil, pipelineOutputs[source])
	}
	return leaves
}

func buildFCAFEvidenceIndex(
	leaves []fcafEvidenceLeaf,
	resolved map[string]any,
	eventIDs map[string]int64,
) []FCAFEvidenceReference {
	index := make([]FCAFEvidenceReference, 0, len(leaves))
	for _, leaf := range leaves {
		value := valueAtPath(resolved[leaf.source], leaf.path)
		digest := ""
		if value != nil {
			if data, err := json.Marshal(value); err == nil {
				sum := sha256.Sum256(data)
				digest = hex.EncodeToString(sum[:])
			}
		}
		for _, ref := range leaf.refs {
			stepID := refStepID(ref)
			index = append(index, FCAFEvidenceReference{
				Source:  leaf.source,
				Path:    formatLeafPath(leaf.path),
				Ref:     ref,
				StepID:  stepID,
				EventID: eventIDs[stepID],
				SHA256:  digest,
			})
		}
	}
	return index
}

func valueAtPath(value any, path []any) any {
	for _, segment := range path {
		switch key := segment.(type) {
		case string:
			m, ok := value.(map[string]any)
			if !ok {
				return nil
			}
			value = m[key]
		case int:
			items, ok := value.([]any)
			if !ok || key >= len(items) {
				return nil
			}
			value = items[key]
		}
	}
	return value
}

func formatLeafPath(path []any) string {
	var b strings.Builder
	for _, segment := range path {
		switch key := segment.(type) {
		case string:
			if b.Len() > 0 {
				b.WriteByte('.')
			}
			b.WriteString(key)
		case int:
			b.WriteString("[" + strconv.Itoa(key) + "]")
		}
	}
	return b.String()
}

// refStepID returns the first dotted segment of an expression's initial value.
func refStepID(ref string) string {
	initial, _, err := pipelineinternal.ParsePipeline(ref)
	if err != nil {
		return ""
	}
	first, _, _ := strings.Cut(initial, ".")
	first, _, _ = strings.Cut(first, "[")
	return first
}

func normalizeValidationTestIDs(payload FCAFValidationActivityInput) []string {
	ids := make([]string, 0, len(payload.TestIDs))
	seen := make(map[string]struct{}, len(payload.TestIDs))
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, id := range payload.TestIDs {
		add(id)
	}
	add(payload.TestID)
	return ids
}

func resolveDefaultFCAFValidationCatalogRoot() string {
	if _, err := os.Stat(DefaultFCAFValidationCatalogRoot); err == nil {
		return DefaultFCAFValidationCatalogRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return DefaultFCAFValidationCatalogRoot
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, DefaultFCAFValidationCatalogRoot)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return DefaultFCAFValidationCatalogRoot
}
