// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { PRIVATE_PATHS, SITE_ORIGIN } from '$lib/seo';

import { baseLocale, locales } from '@/i18n';

/**
 * AI crawlers are named so the intent is explicit: answer engines and models may read,
 * cite and learn from every public page. They share the `*` group's rules (RFC 9309 §2.1),
 * so private areas stay excluded for them too.
 */
const AI_CRAWLERS = [
	'GPTBot',
	'OAI-SearchBot',
	'ChatGPT-User',
	'ClaudeBot',
	'Claude-SearchBot',
	'Claude-User',
	'anthropic-ai',
	'PerplexityBot',
	'Perplexity-User',
	'Google-Extended',
	'GoogleOther',
	'Applebot-Extended',
	'Meta-ExternalAgent',
	'Amazonbot',
	'CCBot',
	'cohere-ai',
	'MistralAI-User',
	'DuckAssistBot',
	'YouBot'
];

export const GET = () => {
	const origin = SITE_ORIGIN;
	const prefixes = ['', ...locales.filter((locale) => locale !== baseLocale).map((l) => `/${l}`)];
	const disallow = prefixes.flatMap((prefix) =>
		PRIVATE_PATHS.map((path) => `Disallow: ${prefix}${path}`)
	);

	const body = [
		'User-agent: *',
		...AI_CRAWLERS.map((agent) => `User-agent: ${agent}`),
		'Allow: /',
		...disallow,
		'',
		`Sitemap: ${origin}/sitemap.xml`,
		'',
		`# LLM-friendly index of this site: ${origin}/llms.txt (full text: ${origin}/llms-full.txt)`,
		''
	].join('\n');

	return new Response(body, {
		headers: {
			'content-type': 'text/plain; charset=utf-8',
			'cache-control': 'public, max-age=86400'
		}
	});
};
