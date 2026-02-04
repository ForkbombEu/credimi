// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflowengine

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"

const (
	MobileRunnerSemaphoreRequestQueued   = mobilerunnersemaphore.MobileRunnerSemaphoreRequestQueued
	MobileRunnerSemaphoreRequestGranted  = mobilerunnersemaphore.MobileRunnerSemaphoreRequestGranted
	MobileRunnerSemaphoreRequestTimedOut = mobilerunnersemaphore.MobileRunnerSemaphoreRequestTimedOut

	MobileRunnerSemaphoreDefaultNamespace = "default"
)
