// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// The harness UI attaches to a chat only while its sessions and workflow status report it open.
// Credimi embeds the UI for a closed chat only when the runner agent answers its
// `authoring_events` query, so until the page's first attach stream ends a closed chat is
// reported open: the harness attaches, the service replays the chat's events through that
// query, and the harness then reads the chat closed, as for a chat that ends while it is open.
// Meanwhile the composer is hidden (embed.css) and messages are refused here and by the service.
(() => {
	const api = new URL('api/', document.baseURI).pathname;
	const root = document.documentElement;
	const fetch = window.fetch.bind(window);
	let attached = false;

	const json = (body, status) =>
		new Response(JSON.stringify(body), {
			status,
			headers: { 'Content-Type': 'application/json' }
		});

	const open = (state) => {
		if (!state?.closed) return state;
		root.classList.add('credimi-replay');
		return { ...state, closed: false };
	};

	const finish = () => {
		attached = true;
		root.classList.remove('credimi-replay');
	};

	window.fetch = async (input, init) => {
		const path = new URL(input instanceof Request ? input.url : String(input), document.baseURI)
			.pathname;
		const method = (init?.method ?? (input instanceof Request ? input.method : 'GET')).toUpperCase();
		if (attached || !path.startsWith(api)) return fetch(input, init);
		const route = path.slice(api.length);
		if (method === 'POST' && (route === 'chat' || route === 'messages')) {
			if (root.classList.contains('credimi-replay')) {
				return json({ error: 'session_closed', message: 'This chat has ended.' }, 409);
			}
			return fetch(input, init);
		}
		if (method !== 'GET') return fetch(input, init);
		const response = await fetch(input, init);
		if (!response.ok) return response;
		if (route === 'sessions') {
			return json((await response.json()).map(open), response.status);
		}
		if (route.startsWith('workflow-status/')) {
			return json(open(await response.json()), response.status);
		}
		if (route === 'attach' && response.body) {
			return new Response(response.body.pipeThrough(new TransformStream({ flush: finish })), {
				status: response.status,
				statusText: response.statusText,
				headers: response.headers
			});
		}
		return response;
	};
})();
