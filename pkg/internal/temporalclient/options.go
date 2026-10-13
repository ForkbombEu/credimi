// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"sync"

	"github.com/forkbombeu/credimi/pkg/utils"
	"go.temporal.io/sdk/client"
	tlog "go.temporal.io/sdk/log"
)

var (
	loggerMu sync.RWMutex
	logger   tlog.Logger
)

// ConfigureLogger sets the logger used by Temporal clients created after the call.
func ConfigureLogger(l tlog.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = l
}

// baseOptions returns the connection options shared by every Temporal client.
// A nil logger is left unset so the SDK keeps its default.
func baseOptions(namespace string) client.Options {
	options := client.Options{
		HostPort:  utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort),
		Namespace: namespace,
	}
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if logger != nil {
		options.Logger = logger
	}
	return options
}
