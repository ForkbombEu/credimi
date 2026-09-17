// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemporalWorkersDisabled(t *testing.T) {
	t.Setenv(TemporalWorkersDisabledEnv, "")
	assert.False(t, TemporalWorkersDisabled())

	for _, value := range []string{"1", "true", "TRUE", " yes ", "Yes"} {
		t.Run("enabled_"+value, func(t *testing.T) {
			t.Setenv(TemporalWorkersDisabledEnv, value)
			assert.True(t, TemporalWorkersDisabled())
		})
	}

	for _, value := range []string{"0", "false", "no", "off"} {
		t.Run("disabled_"+value, func(t *testing.T) {
			t.Setenv(TemporalWorkersDisabledEnv, value)
			assert.False(t, TemporalWorkersDisabled())
		})
	}
}
