// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	commonpb "go.temporal.io/api/common/v1"
)

// DecodeStringPayload decodes a Temporal payload holding a string (such as a
// memo field) through the Credimi data converter. Payloads the converter
// cannot decode fall back to their raw data with surrounding quotes trimmed.
func DecodeStringPayload(p *commonpb.Payload) string {
	if p == nil {
		return ""
	}
	var s string
	if err := temporalcrypto.DataConverter().FromPayload(p, &s); err != nil {
		return strings.Trim(string(p.GetData()), `"`)
	}
	return s
}
