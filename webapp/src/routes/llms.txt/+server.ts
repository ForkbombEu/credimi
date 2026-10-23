// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { SITE_ORIGIN } from '$lib/seo';
import { renderLlmsIndex } from '$lib/seo/llms';
import { listPublicPages } from '$lib/seo/public-pages';

export const GET = async ({ fetch }) => {
	const origin = SITE_ORIGIN;
	return new Response(renderLlmsIndex(await listPublicPages(fetch), origin), {
		headers: {
			'content-type': 'text/markdown; charset=utf-8',
			'cache-control': 'public, max-age=3600'
		}
	});
};
