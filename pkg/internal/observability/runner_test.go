// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package observability

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestStampRunnerIdentity(t *testing.T) {
	owner := core.NewRecord(core.NewBaseCollection("users"))
	owner.Id = "user123"

	forged := &coltracepb.ExportTraceServiceRequest{ResourceSpans: []*tracepb.ResourceSpans{
		{Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			stringKV("service.name", "credimi"),
			stringKV("credimi.owner.id", "someone-else"),
			stringKV("runner_id", "acme/runner"),
		}}},
		{}, // no resource at all
	}}
	body, err := proto.Marshal(forged)
	require.NoError(t, err)

	out, err := stampRunnerIdentity("traces", body, owner)
	require.NoError(t, err)
	var got coltracepb.ExportTraceServiceRequest
	require.NoError(t, proto.Unmarshal(out, &got))
	for _, rs := range got.GetResourceSpans() {
		attrs := map[string][]string{}
		for _, kv := range rs.GetResource().GetAttributes() {
			attrs[kv.GetKey()] = append(attrs[kv.GetKey()], kv.GetValue().GetStringValue())
		}
		require.Equal(t, []string{RunnerServiceName}, attrs["service.name"])
		require.Equal(t, []string{"user123"}, attrs["credimi.owner.id"])
		require.Equal(t, []string{"users"}, attrs["credimi.owner.collection"])
	}
	require.Equal(
		t,
		"acme/runner",
		got.GetResourceSpans()[0].GetResource().GetAttributes()[0].GetValue().GetStringValue(),
	)

	_, err = stampRunnerIdentity("profiles", body, owner)
	require.Error(t, err)
	_, err = stampRunnerIdentity("metrics", []byte{0xff, 0xff}, owner)
	require.Error(t, err)
}

func TestStampRunnerIdentityBoundsResourceGroupsForEverySignal(t *testing.T) {
	owner := core.NewRecord(core.NewBaseCollection("users"))
	owner.Id = "user123"
	body := bytes.Repeat([]byte{0x0a, 0x00}, maxRunnerResourceGroups+1)

	for _, signal := range []string{"traces", "metrics", "logs"} {
		t.Run(signal, func(t *testing.T) {
			_, err := stampRunnerIdentity(signal, body, owner)
			require.ErrorContains(t, err, "too many resource groups")
		})
	}
}

func TestStampRunnerIdentityBoundsExpandedSizeForEverySignal(t *testing.T) {
	owner := core.NewRecord(core.NewBaseCollection("users"))
	owner.Id = strings.Repeat("x", maxRunnerBatchBytes)
	body := []byte{0x0a, 0x00}

	for _, signal := range []string{"traces", "metrics", "logs"} {
		t.Run(signal, func(t *testing.T) {
			_, err := stampRunnerIdentity(signal, body, owner)
			require.ErrorContains(t, err, "stamped batch exceeds size limit")
		})
	}
}
