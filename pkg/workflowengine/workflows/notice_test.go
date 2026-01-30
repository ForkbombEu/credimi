// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Println("IGNORE ERRORS BELOW (expected workflow test logs)")
	code := m.Run()
	fmt.Println("IGNORE ERRORS ABOVE (expected workflow test logs)")
	os.Exit(code)
}
