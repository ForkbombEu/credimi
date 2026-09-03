// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

// The webworker lib reference gives this file ServiceWorkerGlobalScope types
// without switching the rest of the app away from the DOM lib.
// `self` still resolves to DOM's Window under svelte-check, hence the cast.
const sw = self as unknown as ServiceWorkerGlobalScope;

// Plain English strings: paraglide's locale strategies (url/cookie) need
// window/document APIs that are unavailable in the service worker context.
const PUSH_RESULT_TITLE: Record<string, string> = {
	success: 'completed',
	failed: 'failed',
	canceled: 'canceled'
};

const FALLBACK_TITLE = 'finished';

const NOTIFICATION_ICON = '/logos/credimi_logo-transp_emblem.png';

type PushPayload = {
	pipeline_name: string;
	organization?: string;
	result: string;
	duration?: string;
	error?: string;
	url: string;
};

sw.addEventListener('push', (event) => {
	const payload = parsePushPayload(event.data?.text());
	if (!payload?.url) return;

	const title = `${payload.pipeline_name} — ${PUSH_RESULT_TITLE[payload.result] ?? FALLBACK_TITLE}`;

	event.waitUntil(
		sw.registration.showNotification(title, {
			body: buildPushBody(payload),
			icon: NOTIFICATION_ICON,
			tag: payload.url,
			data: { url: payload.url }
		})
	);
});

function buildPushBody(payload: PushPayload): string {
	const context: string[] = [];
	if (payload.organization) context.push(`in ${payload.organization}`);
	if (payload.duration) context.push(`after ${payload.duration}`);

	const lines: string[] = [];
	if (context.length > 0) {
		const sentence = context.join(' ');
		lines.push(sentence.charAt(0).toUpperCase() + sentence.slice(1) + '.');
	}
	if (payload.error) lines.push(payload.error);

	return lines.join('\n');
}

sw.addEventListener('notificationclick', (event) => {
	event.notification.close();

	const url: unknown = event.notification.data?.url;
	if (typeof url !== 'string' || !url) return;

	event.waitUntil(
		sw.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(async (clients) => {
			for (const client of clients) {
				await client.focus();
				if ('navigate' in client) await client.navigate(url);
				return;
			}
			await sw.clients.openWindow(url);
		})
	);
});

function parsePushPayload(raw: string | undefined): PushPayload | null {
	if (!raw) return null;
	try {
		const data = JSON.parse(raw) as Partial<PushPayload>;
		if (typeof data.pipeline_name !== 'string' || typeof data.url !== 'string') return null;
		return {
			pipeline_name: data.pipeline_name,
			organization: typeof data.organization === 'string' ? data.organization : undefined,
			result: data.result ?? '',
			duration: typeof data.duration === 'string' ? data.duration : undefined,
			error: typeof data.error === 'string' ? data.error : undefined,
			url: data.url
		};
	} catch {
		return null;
	}
}
