// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Handle, HandleFetch } from '@sveltejs/kit';

import { sequence } from '@sveltejs/kit/hooks';
import { env } from '$env/dynamic/private';

import { paraglideMiddleware } from '@/i18n/paraglide/server';

const handleParaglide: Handle = ({ event, resolve }) =>
	paraglideMiddleware(event.request, ({ request, locale }) => {
		event.request = request;

		return resolve(event, {
			transformPageChunk: ({ html }) => html.replace('%paraglide.lang%', locale)
		});
	});

const compressibleType = /^(text\/|application\/(json|xml|javascript))/;

// Rendered pages, load data and server endpoints reach the browser through PocketBase's proxy,
// which does not compress them. Static and prerendered files never get here: the adapter serves
// them precompressed.
const handleCompression: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);
	if (
		!response.body ||
		response.headers.has('content-encoding') ||
		!compressibleType.test(response.headers.get('content-type') ?? '') ||
		!/\bgzip\b/.test(event.request.headers.get('accept-encoding') ?? '')
	) {
		return response;
	}

	const headers = new Headers(response.headers);
	headers.set('content-encoding', 'gzip');
	headers.delete('content-length');
	headers.append('vary', 'Accept-Encoding');
	return new Response(response.body.pipeThrough(new CompressionStream('gzip')), {
		status: response.status,
		statusText: response.statusText,
		headers
	});
};

export const handle: Handle = sequence(handleCompression, handleParaglide);

// PocketBase listens beside the webapp (Procfile: localhost:8090). Dev points VITE_API elsewhere.
const pocketbaseURL = env.VITE_API || 'http://localhost:8090';

// The webapp calls PocketBase on its own origin (`/api/...`). On the server that origin is the
// webapp itself, and SvelteKit answers same-origin fetches with its own router, which has no
// `/api` routes. Send those requests to PocketBase instead.
export const handleFetch: HandleFetch = ({ event, request, fetch }) => {
	const url = new URL(request.url);
	if (url.origin !== event.url.origin || !url.pathname.startsWith('/api/')) return fetch(request);
	return fetch(new Request(new URL(url.pathname + url.search, pocketbaseURL), request));
};
