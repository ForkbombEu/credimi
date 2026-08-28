// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

type Status string

const (
	StatusPass          Status = "pass"
	StatusFail          Status = "fail"
	StatusBlocked       Status = "blocked"
	StatusSkipped       Status = "skipped"
	StatusNotApplicable Status = "not_applicable"
	StatusInconclusive  Status = "inconclusive"
	StatusError         Status = "error"
)

type Input struct {
	Value   any
	Bundle  evidence.Bundle
	Params  map[string]any
	Runtime map[string]any
	Suite   dsl.Suite
}

type Result struct {
	Status  Status         `json:"status"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type Validator interface {
	ID() string
	Validate(ctx context.Context, input Input) Result
}

func DecodeParams[T any](params map[string]any) (T, error) {
	var out T
	raw, err := json.Marshal(params)
	if err != nil {
		return out, fmt.Errorf("marshal params: %w", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode params: %w", err)
	}
	return out, nil
}
