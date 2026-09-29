// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// THROWAWAY: same-origin iframe target for /dev/live-view-sheet preview.

import { error, type RequestHandler } from '@sveltejs/kit';
import { dev } from '$app/environment';

export const GET: RequestHandler = () => {
	if (!dev) error(404);

	const html = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Fake live view</title>
<style>
  html, body { margin: 0; height: 100%; }
  body {
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    gap: 0.75rem; background: #ece8f7; color: #1e1b4b; font-family: Inter, system-ui, sans-serif;
  }
  .label { font-size: 0.875rem; font-weight: 500; letter-spacing: 0.08em; text-transform: uppercase; }
  .tick { font-family: "Source Code Pro", ui-monospace, monospace; font-size: 2.5rem; font-variant-numeric: tabular-nums; }
  .hint { font-size: 0.75rem; opacity: 0.7; }
</style>
</head>
<body>
  <p class="label">Fake live view</p>
  <p class="tick" id="tick">0s</p>
  <p class="hint">Same-origin preview stream</p>
  <script>
    let n = 0;
    const el = document.getElementById('tick');
    setInterval(() => { n += 1; el.textContent = n + 's'; }, 1000);
  </script>
</body>
</html>`;

	return new Response(html, {
		headers: {
			'content-type': 'text/html; charset=utf-8',
			'cache-control': 'no-store'
		}
	});
};
