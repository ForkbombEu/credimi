// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getContentBySlug } from '$lib/content';
import { getContentSeo } from '$lib/content/seo';

// static/pages/conformance-automation/en.md keeps the metadata (tags, SEO, social card); the page body lives here.
export async function load({ fetch }) {
	const content = await getContentBySlug('conformance-automation', fetch);
	if (!content) {
		error(404);
	}

	return { seo: getContentSeo(content) };
}
