// SPDX-FileCopyrightText: 2025 Forkbomb BV

// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getLocale, baseLocale } from '@/i18n/index.js';
import { Effect as _, pipe, Either } from 'effect';
import { assets } from '$app/paths';
import { getExceptionMessage } from '@/utils/errors.js';

//

export async function load({ params, fetch }) {
	const importPost = (locale: string) =>
		pipe(
			_.tryPromise({
				try: () => fetch(`${assets}/pages/${params.slug}/${locale}.md`),
				catch: (e) =>
					new FileFetchError(`Page not found ${params.slug}: ${getExceptionMessage(e)}`)
			}),
			_.andThen((response) =>
				_.tryPromise({
					try: () => {
						const contentType = response.headers.get('content-type');
						
						if (!response.ok) {
							
							if (response.status === 404) {
								throw new FileFetchError(`Page not found: ${params.slug.join('/')}`);
							} else {
								throw new FileFetchError(
									`Server error fetching page ${params.slug.join('/')}: ${response.statusText}`
								);
							}
						}
						if (contentType && contentType.toLowerCase().includes('text/html')) {
							throw new FileFetchError(
								`Expected markdown file for page ${params.slug.join('/')}, but received HTML.`
							);
						}
						return response.text()},
					catch: (e) =>
						e
				})
			)
		);

	const post = await pipe(
		importPost(getLocale()),
		_.catchAll(() => importPost(baseLocale)),
		_.either,
		_.runPromise
	);

	if (Either.isLeft(post)) {
		if (post.left instanceof FileFetchError) {
			error(404, post.left.message);
		} else {
			error(500, 'Error validating page import');
		}
	}

	return {
		content: post.right,
		slug: params.slug
	};
}

class FileFetchError extends Error {}
