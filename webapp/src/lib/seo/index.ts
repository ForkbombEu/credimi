// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as staticPublicEnv from '$env/static/public';

import { appName } from '@/brand';
import { baseLocale, deLocalizeHref, localizeHref, locales } from '@/i18n';

/**
 * Public origin for canonical URLs, hreflang, Open Graph, sitemap and llms.txt. Requests reach the
 * webapp through the PocketBase proxy with the internal UI host, so the origin cannot come from the
 * request. Self-hosted builds set PUBLIC_SITE_URL.
 */
// Namespace import: the variable is optional, and a named import of an unset variable fails the build.
const publicEnv: Record<string, string | undefined> = staticPublicEnv;
export const SITE_ORIGIN = new URL(publicEnv.PUBLIC_SITE_URL || 'https://credimi.io').origin;

export const SITE_DESCRIPTION =
	'Open-source conformance and interoperability testing for EUDI wallets, credential issuers and verifiers: OpenID4VCI, OpenID4VP and EUDI FCAF, run on real devices with audit-grade evidence.';

export const DOCS_URL = 'https://docs.credimi.io';
export const ATLAS_URL = 'https://atlas.credimi.io';
export const SOURCE_URL = 'https://github.com/forkbombeu/credimi';

/** 1200×630 social cards under static/og/. */
export const ogImages = {
	default: '/og/default.png',
	conformanceAutomation: '/og/conformance-automation.png',
	hub: '/og/hub.png',
	scoreboard: '/og/scoreboard.png'
} as const;

export type JsonLd = Record<string, unknown>;

/** What a route declares about itself; the root layout turns it into <head> tags. */
export type PageSeo = {
	/** Page title without the site suffix. */
	title?: string;
	description?: string;
	/** Root-relative or absolute image URL. */
	image?: string;
	type?: 'website' | 'article';
	noindex?: boolean;
	article?: { publishedTime: string; modifiedTime: string; tags: string[] };
	jsonLd?: JsonLd[];
};

/** A JSON-LD script tag; `<` is escaped so record data (names, descriptions) cannot close the element. */
export function jsonLdScriptTag(data: JsonLd): string {
	return `<script type="application/ld+json">${JSON.stringify(data).replace(/</g, '\\u003c')}</script>`;
}

/** SEO for fixed public routes; read by their loaders, the sitemap and llms.txt. */
export const staticPages = {
	'/': {
		title: 'EUDI wallet conformance and interoperability testing',
		description: SITE_DESCRIPTION,
		image: ogImages.default
	},
	'/hub': {
		title: 'Hub: EUDI wallets, issuers, verifiers and conformance checks',
		description:
			'Try live EUDI wallets, credential issuers, verifiers and credentials, and browse OpenID4VCI, OpenID4VP and FCAF conformance check suites. Every entry exposes a working flow.',
		image: ogImages.hub
	},
	'/scoreboard': {
		title: 'Scoreboard: EUDI conformance and interoperability results',
		description:
			'Public results of published Credimi pipelines: which EUDI wallets, credential issuers and verifiers pass issuance, presentation and conformance checks, and when they last ran.',
		image: ogImages.scoreboard
	},
	'/news': {
		title: 'News: EUDI standards and conformance updates',
		description:
			'Changes in EUDI Wallet specifications, OpenID4VC drafts and conformance suites, and what they mean for wallets, credential issuers and verifiers tested on Credimi.',
		image: ogImages.default
	},
	'/organizations': {
		title: 'Organizations building on EUDI',
		description:
			'Organizations that publish EUDI wallets, credential issuers, verifiers and conformance pipelines on the Credimi Hub.',
		image: ogImages.default
	}
} satisfies Record<string, PageSeo>;

/** URL paths crawlers should skip; mirrors NOINDEX_ROUTE_PREFIXES below, which works on route ids. */
export const PRIVATE_PATHS = [
	'/my/',
	'/api/',
	'/_/',
	'/temporal-ui/',
	'/login',
	'/register',
	'/forgot-password',
	'/reset-password',
	'/keypairoom',
	'/logout',
	'/verify-email',
	'/ui-tests',
	'/pipeline-report',
	'/pipeline-result',
	'/tests/wallet/',
	'/pages/tags'
];

export function absoluteUrl(pathOrUrl: string, origin: string): string {
	return new URL(pathOrUrl, origin).toString();
}

/** Every locale variant of a path, keyed by hreflang, plus x-default. */
export function localeAlternates(
	path: string,
	origin: string
): { hreflang: string; href: string }[] {
	const basePath = deLocalizeHref(path);
	return [
		...locales.map((locale) => ({
			hreflang: locale,
			href: absoluteUrl(localizeHref(basePath, { locale }), origin)
		})),
		{
			hreflang: 'x-default',
			href: absoluteUrl(localizeHref(basePath, { locale: baseLocale }), origin)
		}
	];
}

/** Plain text from stored HTML, cut at a word boundary: for meta descriptions and llms.txt. */
export function plainText(html: string | null | undefined, maxLength = 160): string {
	const text = (html ?? '')
		.replace(/<[^>]*>/g, ' ')
		.replace(/&nbsp;/g, ' ')
		.replace(/&amp;/g, '&')
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&#39;/g, "'")
		.replace(/\s+/g, ' ')
		.trim();
	if (text.length <= maxLength) return text;
	const cut = text.slice(0, maxLength - 1);
	return `${cut.slice(0, cut.lastIndexOf(' ') > 0 ? cut.lastIndexOf(' ') : cut.length)}…`;
}

/** Route ids that must never be indexed: private areas, auth, and ephemeral run views. */
const NOINDEX_ROUTE_PREFIXES = [
	'/my',
	'/(nru)',
	'/keypairoom',
	'/logout',
	'/ui-tests',
	'/verify-email',
	'/(public)/pipeline-report',
	'/(public)/pipeline-result',
	'/(public)/tests/wallet',
	'/(public)/pages/tags'
];

export function isIndexableRoute(routeId: string | null): boolean {
	if (!routeId) return false;
	return !NOINDEX_ROUTE_PREFIXES.some((prefix) => routeId.startsWith(prefix));
}

export function organizationJsonLd(origin: string): JsonLd {
	return {
		'@context': 'https://schema.org',
		'@type': 'Organization',
		'@id': `${origin}/#organization`,
		name: appName,
		url: origin,
		logo: absoluteUrl('/logo.svg', origin),
		description: SITE_DESCRIPTION,
		sameAs: [SOURCE_URL, DOCS_URL, ATLAS_URL],
		parentOrganization: {
			'@type': 'Organization',
			name: 'Forkbomb BV',
			url: 'https://forkbomb.eu'
		}
	};
}

export function websiteJsonLd(origin: string): JsonLd {
	return {
		'@context': 'https://schema.org',
		'@type': 'WebSite',
		'@id': `${origin}/#website`,
		name: appName,
		url: origin,
		description: SITE_DESCRIPTION,
		publisher: { '@id': `${origin}/#organization` },
		inLanguage: [...locales]
	};
}

export function softwareApplicationJsonLd(origin: string): JsonLd {
	return {
		'@context': 'https://schema.org',
		'@type': 'SoftwareApplication',
		name: appName,
		url: origin,
		description: SITE_DESCRIPTION,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web, Android, iOS',
		license: 'https://www.gnu.org/licenses/agpl-3.0.html',
		isAccessibleForFree: true,
		publisher: { '@id': `${origin}/#organization` },
		image: absoluteUrl(ogImages.default, origin),
		featureList: [
			'OpenID4VCI and OpenID4VP conformance testing',
			'EUDI FCAF test cases with hashed evidence',
			'Wallet UI automation on real Android devices, emulators and iOS simulators',
			'Issuer to wallet to verifier interoperability pipelines',
			'CI integration for wallets, issuers and verifiers',
			'Public Hub of live issuers, verifiers and wallets'
		]
	};
}
