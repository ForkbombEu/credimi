// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInteropStatusFromRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rate float64
		want interopStatus
	}{
		{rate: 90, want: interopStatusStable},
		{rate: 89.9, want: interopStatusFlaky},
		{rate: 70, want: interopStatusFlaky},
		{rate: 69.9, want: interopStatusFailing},
		{rate: 50, want: interopStatusFailing},
		{rate: 49.9, want: interopStatusBroken},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%g", tt.rate), func(t *testing.T) {
			t.Parallel()
			got := interopStatusFromRate(tt.rate)
			require.Equal(t, tt.want, got)
		})
	}
}
