// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflowengine

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"

const (
	MobileRunnerSemaphoreRunQueued   = mobilerunnersemaphore.MobileRunnerSemaphoreRunQueued
	MobileRunnerSemaphoreRunStarting = mobilerunnersemaphore.MobileRunnerSemaphoreRunStarting
	MobileRunnerSemaphoreRunRunning  = mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning
	MobileRunnerSemaphoreRunFailed   = mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed
	MobileRunnerSemaphoreRunCanceled = mobilerunnersemaphore.MobileRunnerSemaphoreRunCanceled
	MobileRunnerSemaphoreRunNotFound = mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound

	MobileRunnerSemaphoreDefaultNamespace = "default"
)
