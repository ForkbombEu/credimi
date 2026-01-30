// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

const port = Number(process.env.MOCK_POCKETBASE_PORT ?? 8090);
const origin = process.env.MOCK_POCKETBASE_ORIGIN ?? '*';

const headers = {
	'Access-Control-Allow-Origin': origin,
	'Access-Control-Allow-Methods': 'GET, OPTIONS',
	'Access-Control-Allow-Headers': '*'
};

const featuresPayload = {
	page: 1,
	perPage: 200,
	totalPages: 1,
	totalItems: 1,
	items: [{ id: 'auth-feature', name: 'auth', active: true }]
};

Bun.serve({
	port,
	fetch(req) {
		const url = new URL(req.url);
		if (req.method === 'OPTIONS') {
			return new Response(null, { status: 204, headers });
		}
		if (
			url.pathname === '/api/collections/features/records' ||
			url.pathname === '/api/collections/features/records/'
		) {
			return new Response(JSON.stringify(featuresPayload), {
				status: 200,
				headers: {
					...headers,
					'Content-Type': 'application/json'
				}
			});
		}
		return new Response('Not Found', { status: 404, headers });
	}
});

console.log(`Mock PocketBase listening on http://127.0.0.1:${port}`);
