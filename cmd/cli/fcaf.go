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
	DetailsFile string
	Screenshots []string
}

type fcafReport struct {
	GeneratedAt time.Time
	Runs        []fcafRun
	Passed      int
	Failed      int
	Blocked     int
	Errors      int
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
	run.Flags().StringVarP(&instanceURL, "instance", "i", "http://localhost:8090", "URL of the Credimi instance")
	cmd.AddCommand(run)
	return cmd
}

func runFCAF(ctx context.Context) error {
	if strings.TrimSpace(apiKey) == "" {
		apiKey = strings.TrimSpace(os.Getenv("CREDIMI_API_KEY"))
	}
	if apiKey == "" || apiKey == "replace-with-local-api-key" {
		return fmt.Errorf("CREDIMI_API_KEY is required; generate a local key and set it in .env or use --api-key")
	}
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
	report := fcafReport{}
	for _, path := range paths {
		if fcafFilter != "" && !strings.Contains(filepath.Base(path), fcafFilter) {
			continue
		}
		run, err := runOneFCAF(ctx, token, orgID, org, path)
		if err != nil {
			run.Status, run.Error = "error", err.Error()
		}
		switch run.Status {
		case "completed", "success", "passed":
			report.Passed++
		case "failed", "canceled":
			report.Failed++
		case "queued", "starting", "running":
			report.Blocked++
		default:
			report.Errors++
		}
		runs = append(runs, run)
		b, _ := json.MarshalIndent(run, "", "  ")
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".json"
		if err := os.WriteFile(filepath.Join(fcafOutput, name), b, 0o644); err != nil {
			return err
		}
		fmt.Printf("%-8s %s\n", run.Status, run.Name)
	}
	return writeFCAFReport(fcafReport{GeneratedAt: time.Now(), Runs: runs, Passed: report.Passed, Failed: report.Failed, Blocked: report.Blocked, Errors: report.Errors})
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
				run.Screenshots, run.DetailsFile = collectFCAFImages(ctx, token, queued.RunURL, fcafOutput, run.Name)
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

func collectFCAFImages(ctx context.Context, token, detailsURL, output, runName string) ([]string, string) {
	status, body, err := getAbsolute(ctx, token, detailsURL)
	if err != nil || status < 200 || status >= 300 {
		return nil, ""
	}
	detailsFile := strings.TrimSuffix(filepath.Base(runName), filepath.Ext(runName)) + ".details.json"
	if err := os.WriteFile(filepath.Join(output, detailsFile), body, 0o644); err != nil {
		detailsFile = ""
	}
	var value any
	if json.Unmarshal(body, &value) != nil {
		return nil, detailsFile
	}
	urls := make([]string, 0)
	collectStrings(value, &urls)
	result := make([]string, 0)
	imageNumber := 0
	for _, imageURL := range urls {
		lower := strings.ToLower(imageURL)
		if !strings.Contains(lower, ".png") && !strings.Contains(lower, ".jpg") && !strings.Contains(lower, ".jpeg") {
			continue
		}
		status, data, err := getAbsolute(ctx, token, imageURL)
		if err != nil || status < 200 || status >= 300 {
			continue
		}
		name := fmt.Sprintf("%s-evidence-%03d%s", strings.TrimSuffix(filepath.Base(runName), filepath.Ext(runName)), imageNumber, filepath.Ext(imageURL))
		if err := os.WriteFile(filepath.Join(output, name), data, 0o644); err == nil {
			result = append(result, name)
			imageNumber++
		}
	}
	return result, detailsFile
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

func writeFCAFReport(report fcafReport) error {
	f, err := os.Create(filepath.Join(fcafOutput, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	const page = `<!doctype html><html><head><meta charset="utf-8"><title>FCAF run report</title><style>body{font:14px sans-serif;line-height:1.4;margin:2rem;color:#202124}h1{margin-bottom:.25rem}.summary{display:flex;gap:1rem;margin:1rem 0}.summary span{border:1px solid #ccc;border-radius:4px;padding:.5rem .75rem}.passed{color:#137333}.failed,.error{color:#b3261e}.blocked{color:#8a4b00}table{border-collapse:collapse;width:100%;table-layout:fixed}td,th{border:1px solid #d8d8d8;padding:.6rem;text-align:left;vertical-align:top}th:nth-child(1){width:24%}th:nth-child(2){width:9%}th:nth-child(3){width:13%}th:nth-child(4){width:26%}th:nth-child(5){width:28%}img{max-width:180px;max-height:140px;object-fit:contain;border:1px solid #ddd;margin:.25rem}code{word-break:break-all}</style></head><body><h1>FCAF pipeline run report</h1><p>Generated {{.GeneratedAt}}</p><div class="summary"><span class="passed">Passed: {{.Passed}}</span><span class="failed">Failed: {{.Failed}}</span><span class="blocked">Blocked: {{.Blocked}}</span><span class="error">Errors: {{.Errors}}</span></div><table><tr><th>Pipeline</th><th>Status</th><th>Duration</th><th>Run details</th><th>Evidence</th></tr>{{range .Runs}}<tr><td><strong>{{.Name}}</strong><br><code>{{.Path}}</code></td><td class="{{statusClass .Status}}">{{.Status}}</td><td>{{duration .StartedAt .FinishedAt}}</td><td>{{if .Error}}<div class="error">{{.Error}}</div>{{end}}{{if .DetailsFile}}<a href="{{.DetailsFile}}">details JSON</a><br>{{end}}{{if .DetailsURL}}<a href="{{.DetailsURL}}">Credimi run</a>{{end}}</td><td>{{if .Screenshots}}{{range .Screenshots}}<a href="{{.}}"><img src="{{.}}" alt="evidence screenshot"></a>{{end}}{{else}}No downloaded screenshots{{end}}</td></tr>{{end}}</table></body></html>`
	funcs := template.FuncMap{"statusClass": func(status string) string {
		if status == "completed" || status == "success" || status == "passed" {
			return "passed"
		}
		if status == "failed" || status == "canceled" {
			return "failed"
		}
		if status == "error" {
			return "error"
		}
		return "blocked"
	}, "duration": func(start, finish time.Time) string {
		if finish.IsZero() {
			return "-"
		}
		return finish.Sub(start).Round(time.Second).String()
	}}
	return template.Must(template.New("fcaf").Funcs(funcs).Parse(page)).Execute(f, report)
}
