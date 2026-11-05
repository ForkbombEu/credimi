// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Loaded with the first page: no OpenTelemetry code here. The SDK is a separate
// chunk imported once the app has started; errors raised before it arrives are
// buffered and sent when it does.

import type { Attributes } from '@opentelemetry/api';

type Report = (error: unknown, attributes: Attributes) => void;

const MAX_PENDING = 50;

let report: Report | undefined;
let context: () => Attributes = () => ({});
const pending: Array<[unknown, Attributes]> = [];

/** Records error as an exception span, with the current page and user. */
export function reportError(error: unknown, attributes: Attributes = {}) {
	// PocketBase rejects superseded requests on purpose.
	if (typeof error === 'object' && error !== null && 'isAbort' in error && error.isAbort) return;
	const all = { 'url.path': location.pathname, ...context(), ...attributes };
	if (report) report(error, all);
	else if (pending.length < MAX_PENDING) pending.push([error, all]);
}

/**
 * Starts the OpenTelemetry browser SDK: fetch spans that carry traceparent to
 * the API, and exception spans for uncaught errors and rejections.
 */
export function startBrowserTelemetry(version: string, attributes: () => Attributes) {
	context = attributes;
	addEventListener('error', (event) =>
		reportError(event.error ?? event.message, { 'exception.source': 'window.error' })
	);
	addEventListener('unhandledrejection', (event) =>
		reportError(event.reason, { 'exception.source': 'unhandledrejection' })
	);
	import('./browser.js')
		.then(({ start }) => {
			report = start(version);
			for (const [error, all] of pending.splice(0)) report(error, all);
		})
		.catch((error) => console.warn('browser telemetry unavailable', error));
}
