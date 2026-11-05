// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package observability

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// RunnerTelemetryPath receives OTLP/protobuf traces, metrics and logs from
// credimi-runner, authenticated with the runner's Credimi API key. Credimi
// forwards them with the collector credentials, so runners hold none.
const RunnerTelemetryPath = "/api/otlp/runner/v1/{signal}"

// RunnerServiceName is the service.name every forwarded batch is given.
const RunnerServiceName = "credimi-runner"

const (
	maxRunnerBatchBytes     = 4 << 20
	maxRunnerResourceGroups = 4096
	// Traces and logs every few seconds, metrics every minute, per runner.
	runnerRate  = rate.Limit(5)
	runnerBurst = 50
)

// stampRunnerIdentity sets service.name and the key owner on every resource of
// an OTLP/protobuf export request, replacing whatever the sender claimed: a
// runner can report only as the user whose key it holds.
func stampRunnerIdentity(signal string, body []byte, owner *core.Record) ([]byte, error) {
	switch signal {
	case "traces", "metrics", "logs":
	default:
		return nil, fmt.Errorf("unknown signal %q", signal)
	}

	forced := []*commonpb.KeyValue{
		stringKV("service.name", RunnerServiceName),
		stringKV("credimi.owner.id", owner.Id),
		stringKV("credimi.owner.collection", owner.Collection().Name),
	}
	groupCount, err := countRunnerResourceGroups(body)
	if err != nil {
		return nil, err
	}
	if groupCount > maxRunnerResourceGroups {
		return nil, fmt.Errorf("too many resource groups: %d", groupCount)
	}
	if !runnerStampedSizeWithinBudget(len(body), groupCount, forced) {
		return nil, fmt.Errorf("stamped batch exceeds size limit")
	}

	var msg proto.Message
	var resources []*resourcepb.Resource
	switch signal {
	case "traces":
		req := &coltracepb.ExportTraceServiceRequest{}
		if err := proto.Unmarshal(body, req); err != nil {
			return nil, err
		}
		for _, rs := range req.GetResourceSpans() {
			if rs.GetResource() == nil {
				rs.Resource = &resourcepb.Resource{}
			}
			resources = append(resources, rs.GetResource())
		}
		msg = req
	case "metrics":
		req := &colmetricspb.ExportMetricsServiceRequest{}
		if err := proto.Unmarshal(body, req); err != nil {
			return nil, err
		}
		for _, rm := range req.GetResourceMetrics() {
			if rm.GetResource() == nil {
				rm.Resource = &resourcepb.Resource{}
			}
			resources = append(resources, rm.GetResource())
		}
		msg = req
	case "logs":
		req := &collogspb.ExportLogsServiceRequest{}
		if err := proto.Unmarshal(body, req); err != nil {
			return nil, err
		}
		for _, rl := range req.GetResourceLogs() {
			if rl.GetResource() == nil {
				rl.Resource = &resourcepb.Resource{}
			}
			resources = append(resources, rl.GetResource())
		}
		msg = req
	}
	for _, r := range resources {
		kept := make([]*commonpb.KeyValue, 0, len(r.GetAttributes())+len(forced))
		for _, kv := range r.GetAttributes() {
			if !isForcedKey(kv.GetKey(), forced) {
				kept = append(kept, kv)
			}
		}
		kept = append(kept, forced...)
		r.Attributes = kept
	}
	return proto.Marshal(msg)
}

func countRunnerResourceGroups(body []byte) (int, error) {
	count := 0
	for len(body) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(body)
		if tagLen < 0 {
			return 0, protowire.ParseError(tagLen)
		}
		valueLen := protowire.ConsumeFieldValue(num, typ, body[tagLen:])
		if valueLen < 0 {
			return 0, protowire.ParseError(valueLen)
		}
		if num == 1 && typ == protowire.BytesType {
			count++
			if count > maxRunnerResourceGroups {
				return count, nil
			}
		}
		body = body[tagLen+valueLen:]
	}
	return count, nil
}

func runnerStampedSizeWithinBudget(bodySize, groupCount int, forced []*commonpb.KeyValue) bool {
	resourceSize := proto.Size(&resourcepb.Resource{Attributes: forced})
	groupGrowth := resourceSize +
		protowire.SizeTag(1) +
		protowire.SizeBytes(resourceSize) +
		2*protowire.SizeVarint(^uint64(0))
	return int64(bodySize)+int64(groupCount)*int64(groupGrowth) <= maxRunnerBatchBytes
}

func stringKV(key, value string) *commonpb.KeyValue {
	return &commonpb.KeyValue{
		Key:   key,
		Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: value}},
	}
}

func isForcedKey(key string, forced []*commonpb.KeyValue) bool {
	for _, kv := range forced {
		if kv.GetKey() == key {
			return true
		}
	}
	return false
}

// RunnerTelemetryHandler forwards runner OTLP batches. It must sit behind the
// API key middleware, which sets the key owner as e.Auth.
func RunnerTelemetryHandler() func(*core.RequestEvent) error {
	base := strings.TrimRight(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "/")
	headers := otlpHeaders(os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))
	limiter := &ipLimiter{
		limiters: map[string]*rate.Limiter{},
		rate:     runnerRate,
		burst:    runnerBurst,
	}
	client := &http.Client{Timeout: 15 * time.Second}

	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.UnauthorizedError("", nil)
		}
		if !limiter.allow(e.Auth.Id) {
			return e.TooManyRequestsError("", nil)
		}
		if !strings.HasPrefix(e.Request.Header.Get("Content-Type"), "application/x-protobuf") {
			return e.Error(http.StatusUnsupportedMediaType, "OTLP/protobuf only", nil)
		}
		if enc := e.Request.Header.Get("Content-Encoding"); enc != "" && enc != "identity" {
			return e.Error(http.StatusUnsupportedMediaType, "uncompressed OTLP only", nil)
		}
		body, err := io.ReadAll(
			http.MaxBytesReader(e.Response, e.Request.Body, maxRunnerBatchBytes),
		)
		if err != nil {
			return e.Error(http.StatusRequestEntityTooLarge, "batch too large", nil)
		}
		signal := e.Request.PathValue("signal")
		body, err = stampRunnerIdentity(signal, body, e.Auth)
		if err != nil {
			return e.BadRequestError("invalid OTLP/protobuf "+signal, nil)
		}

		req, err := http.NewRequestWithContext(
			e.Request.Context(),
			http.MethodPost,
			base+"/v1/"+signal,
			bytes.NewReader(body),
		)
		if err != nil {
			return err
		}
		req.Header = headers.Clone()
		req.Header.Set("Content-Type", "application/x-protobuf")
		resp, err := client.Do(req)
		if err != nil {
			return e.Error(http.StatusBadGateway, "collector unreachable", nil)
		}
		defer resp.Body.Close()
		return e.Stream(resp.StatusCode, "application/x-protobuf", resp.Body)
	}
}
