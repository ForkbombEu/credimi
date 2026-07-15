// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	fcafDir      string
	fcafOutput   string
	fcafFilter   string
	fcafTimeout  time.Duration
	fcafInterval time.Duration
)

type fcafRun struct {
	Name        string
	Path        string
	Status      string
	Error       string
	StartedAt   time.Time
	FinishedAt  time.Time
	DetailsURL  string
	Screenshots []string
}

// NewFCAFCommand runs the reusable FCAF pipeline YAML files sequentially.
func NewFCAFCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fcaf",
		Short: "Run FCAF wallet pipelines and collect evidence",
	}
	run := &cobra.Command{
		Use:   "run",
		Short: "Run FCAF pipeline YAML files sequentially",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFCAF(cmd.Context())
		},
	}
	run.Flags().StringVar(&fcafDir, "dir", "config_templates/fcaf/wallet_solution/relying_party/pipelines", "directory containing pipeline YAML files")
	run.Flags().StringVar(&fcafOutput, "output", "fcaf-report", "directory for JSON, screenshots, and HTML output")
	run.Flags().StringVar(&fcafFilter, "filter", "", "run only files whose name contains this value")
	run.Flags().DurationVar(&fcafTimeout, "timeout", 30*time.Minute, "maximum time allowed for each pipeline")
	run.Flags().DurationVar(&fcafInterval, "interval", 5*time.Second, "queue polling interval")
	run.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication")
	_ = run.MarkFlagRequired("api-key")
	run.Flags().StringVarP(&instanceURL, "instance", "i", "http://localhost:8090", "URL of the Credimi instance")
	cmd.AddCommand(run)
	return cmd
}

func runFCAF(ctx context.Context) error {
	token, err := authenticate(ctx)
	if err != nil {
		return err
	}
	orgID, org, err := getMyOrganization(ctx, token)
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(fcafDir, "*.yaml"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return fmt.Errorf("no YAML pipelines found in %s", fcafDir)
	}
	if err := os.MkdirAll(fcafOutput, 0o755); err != nil {
		return err
	}
	runs := make([]fcafRun, 0, len(paths))
	for _, path := range paths {
		if fcafFilter != "" && !strings.Contains(filepath.Base(path), fcafFilter) {
			continue
		}
		run, err := runOneFCAF(ctx, token, orgID, org, path)
		if err != nil {
			run.Status, run.Error = "error", err.Error()
		}
		runs = append(runs, run)
		b, _ := json.MarshalIndent(run, "", "  ")
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".json"
		if err := os.WriteFile(filepath.Join(fcafOutput, name), b, 0o644); err != nil {
			return err
		}
		fmt.Printf("%-8s %s\n", run.Status, run.Name)
	}
	return writeFCAFReport(runs)
}

func runOneFCAF(ctx context.Context, token, orgID, org, path string) (fcafRun, error) {
	started := time.Now()
	run := fcafRun{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Path: path, StartedAt: started}
	input, err := os.ReadFile(path)
	if err != nil {
		return run, err
	}
	parsed, err := parsePipelineName(input)
	if err != nil {
		return run, err
	}
	run.Name = parsed
	rec, err := findOrCreatePipeline(ctx, token, orgID, &PipelineCLIInput{Name: parsed, YAML: string(input)})
	if err != nil {
		return run, err
	}
	identifier := fmt.Sprintf("%s/%s", org, rec["canonified_name"])
	body, _ := json.Marshal(map[string]string{"pipeline_identifier": identifier, "yaml": string(input)})
	status, response, err := postPipelineRequest(ctx, token, "queue", body)
	if err != nil {
		return run, err
	}
	if status != http.StatusOK {
		return run, fmt.Errorf("queue returned HTTP %d: %s", status, response)
	}
	var queued pipelineQueueResponse
	if err := json.Unmarshal(response, &queued); err != nil {
		return run, err
	}
	run.DetailsURL = queued.RunURL
	if queued.TicketID == "" {
		return run, fmt.Errorf("queue returned no ticket_id")
	}
	deadline := time.NewTimer(fcafTimeout)
	defer deadline.Stop()
	for {
		if queued.Status == "completed" || queued.Status == "failed" || queued.Status == "canceled" || queued.Status == "not_found" {
			run.Status = queued.Status
			run.Error = queued.ErrorMessage
			run.FinishedAt = time.Now()
			if queued.RunURL != "" {
				run.Screenshots = collectFCAFImages(ctx, token, queued.RunURL, fcafOutput)
			}
			return run, nil
		}
		select {
		case <-ctx.Done():
			return run, ctx.Err()
		case <-deadline.C:
			return run, fmt.Errorf("timed out waiting for ticket %s", queued.TicketID)
		case <-time.After(fcafInterval):
		}
		query := url.Values{}
		query.Set("runner_ids", strings.Join(queued.RunnerIDs, ","))
		status, response, err = getPipeline(ctx, token, "queue/"+queued.TicketID+"?"+query.Encode())
		if err != nil {
			return run, err
		}
		if status != http.StatusOK {
			return run, fmt.Errorf("queue status returned HTTP %d: %s", status, response)
		}
		if err := json.Unmarshal(response, &queued); err != nil {
			return run, err
		}
		if queued.RunURL != "" {
			run.DetailsURL = queued.RunURL
		}
	}
}

func getPipeline(ctx context.Context, token, path string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, utils.JoinURL(instanceURL, "api", "pipeline", path), nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, b, err
}

func collectFCAFImages(ctx context.Context, token, detailsURL, output string) []string {
	status, body, err := getAbsolute(ctx, token, detailsURL)
	if err != nil || status < 200 || status >= 300 {
		return nil
	}
	var value any
	if json.Unmarshal(body, &value) != nil {
		return nil
	}
	urls := make([]string, 0)
	collectStrings(value, &urls)
	result := make([]string, 0)
	for i, imageURL := range urls {
		lower := strings.ToLower(imageURL)
		if !strings.Contains(lower, ".png") && !strings.Contains(lower, ".jpg") && !strings.Contains(lower, ".jpeg") {
			continue
		}
		status, data, err := getAbsolute(ctx, token, imageURL)
		if err != nil || status < 200 || status >= 300 {
			continue
		}
		name := fmt.Sprintf("evidence-%03d%s", i, filepath.Ext(imageURL))
		if err := os.WriteFile(filepath.Join(output, name), data, 0o644); err == nil {
			result = append(result, name)
		}
	}
	return result
}

func collectStrings(value any, out *[]string) {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "http") {
			*out = append(*out, v)
		}
	case []any:
		for _, item := range v {
			collectStrings(item, out)
		}
	case map[string]any:
		for _, item := range v {
			collectStrings(item, out)
		}
	}
}

func getAbsolute(ctx context.Context, token, target string) (int, []byte, error) {
	if !strings.HasPrefix(target, "http") {
		target = utils.JoinURL(instanceURL, strings.TrimPrefix(target, "/"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, b, err
}

func writeFCAFReport(runs []fcafRun) error {
	f, err := os.Create(filepath.Join(fcafOutput, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	const page = `<!doctype html><meta charset="utf-8"><title>FCAF run</title><style>body{font:14px sans-serif;margin:2rem}table{border-collapse:collapse;width:100%}td,th{border:1px solid #ccc;padding:.5rem;text-align:left}.ok{color:green}.bad{color:#b00}img{max-width:240px;max-height:180px}</style><h1>FCAF pipeline run</h1><table><tr><th>Pipeline</th><th>Status</th><th>Error</th><th>Evidence</th></tr>{{range .}}<tr><td>{{.Name}}</td><td class="{{.Status}}">{{.Status}}</td><td>{{.Error}}</td><td>{{range .Screenshots}}<a href="{{.}}"><img src="{{.}}"></a> {{end}}{{if .DetailsURL}}<a href="{{.DetailsURL}}">details</a>{{end}}</td></tr>{{end}}</table>`
	return template.Must(template.New("fcaf").Parse(page)).Execute(f, runs)
}
