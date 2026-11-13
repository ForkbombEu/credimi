// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DashboardCodeTTL is how long a dashboard sign-in code can be redeemed.
const DashboardCodeTTL = 60 * time.Second

// maxDashboardCodes bounds the pending codes this process keeps.
const maxDashboardCodes = 4096

// ErrTooManyDashboardCodes is returned when maxDashboardCodes unexpired codes
// are pending.
var ErrTooManyDashboardCodes = errors.New("too many pending dashboard sign-in codes")

// DashboardCode is a pending sign-in to a runner's dashboard: the user with
// record UserID in AuthCollection may sign in to the dashboard of the runner
// with record ID RunnerID until Expires. ReturnTo is the dashboard callback
// the code was sent to; only that dashboard may redeem it.
type DashboardCode struct {
	RunnerID       string
	UserID         string
	AuthCollection string
	ReturnTo       string
	Expires        time.Time
}

// dashboardCodes holds the pending codes of this process, by code. A restart
// drops them: users then sign in again.
var dashboardCodes = struct {
	sync.Mutex
	codes map[string]DashboardCode
}{codes: map[string]DashboardCode{}}

// IssueDashboardCode stores c under a new random code and returns the code.
func IssueDashboardCode(c DashboardCode, now time.Time) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate dashboard code: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(raw)

	dashboardCodes.Lock()
	defer dashboardCodes.Unlock()
	if len(dashboardCodes.codes) >= maxDashboardCodes {
		for k, v := range dashboardCodes.codes {
			if !now.Before(v.Expires) {
				delete(dashboardCodes.codes, k)
			}
		}
		if len(dashboardCodes.codes) >= maxDashboardCodes {
			return "", ErrTooManyDashboardCodes
		}
	}
	dashboardCodes.codes[code] = c
	return code, nil
}

// RedeemDashboardCode removes code and returns its sign-in when it had not
// expired at now. A code can be redeemed once.
func RedeemDashboardCode(code string, now time.Time) (DashboardCode, bool) {
	dashboardCodes.Lock()
	c, ok := dashboardCodes.codes[code]
	delete(dashboardCodes.codes, code)
	dashboardCodes.Unlock()
	if !ok || !now.Before(c.Expires) {
		return DashboardCode{}, false
	}
	return c, true
}
