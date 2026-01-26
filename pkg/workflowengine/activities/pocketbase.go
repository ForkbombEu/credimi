// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"errors"
	"sync"

	"github.com/pocketbase/pocketbase/core"
)

var pocketBaseApp core.App
var pocketBaseAppMu sync.RWMutex

func SetPocketBaseApp(app core.App) {
	pocketBaseAppMu.Lock()
	defer pocketBaseAppMu.Unlock()
	pocketBaseApp = app
}

func getPocketBaseApp() (core.App, error) {
	pocketBaseAppMu.RLock()
	defer pocketBaseAppMu.RUnlock()
	if pocketBaseApp == nil {
		return nil, errors.New("pocketbase app not configured")
	}
	return pocketBaseApp, nil
}
