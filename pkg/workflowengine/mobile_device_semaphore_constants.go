// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflowengine

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"

const (
	MobileDeviceSemaphoreRunQueued   = mobiledevicesemaphore.MobileDeviceSemaphoreRunQueued
	MobileDeviceSemaphoreRunStarting = mobiledevicesemaphore.MobileDeviceSemaphoreRunStarting
	MobileDeviceSemaphoreRunRunning  = mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning
	MobileDeviceSemaphoreRunFailed   = mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed
	MobileDeviceSemaphoreRunCanceled = mobiledevicesemaphore.MobileDeviceSemaphoreRunCanceled
	MobileDeviceSemaphoreRunNotFound = mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound

	MobileDeviceSemaphoreDefaultNamespace = "default"
)
