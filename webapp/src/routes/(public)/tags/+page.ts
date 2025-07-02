// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getContentBySlug, URL_SEARCH_PARAM_NAME } from '$lib/content';
import type { Tag } from '$lib/content/tags-i18n';
import tagsIndex from '$lib/content/tags-list.generated.json';
import { error } from '@sveltejs/kit';

export const load = async ({ url }) => {
	const paramTag = url.searchParams.get(URL_SEARCH_PARAM_NAME);

	if (!paramTag || !(paramTag in tagsIndex)) {
		error(404);
	}

	const contentPages = (
		await Promise.all(tagsIndex[paramTag as Tag].map((slug) => getContentBySlug(slug)))
	).filter((item) => item !== undefined);

	return { pages: contentPages, tag: paramTag };
};
