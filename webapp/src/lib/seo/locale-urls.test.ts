// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { canonicalUrl, localeAlternates } from './index';

const origin = 'https://credimi.io';

describe('locale URLs of a page written in English only', () => {
	const available = ['en'];

	it('lists only the English page and x-default as alternates', () => {
		expect(localeAlternates('/it/pages/fcaf-conformance', origin, available)).toEqual([
			{ hreflang: 'en', href: 'https://credimi.io/pages/fcaf-conformance' },
			{ hreflang: 'x-default', href: 'https://credimi.io/pages/fcaf-conformance' }
		]);
	});

	it('makes the English URL canonical for the Italian fallback', () => {
		expect(canonicalUrl('/it/pages/fcaf-conformance', origin, 'it', available)).toBe(
			'https://credimi.io/pages/fcaf-conformance'
		);
	});

	it('keeps the English URL canonical for itself', () => {
		expect(canonicalUrl('/pages/fcaf-conformance', origin, 'en', available)).toBe(
			'https://credimi.io/pages/fcaf-conformance'
		);
	});
});

describe('locale URLs of a translated page', () => {
	it('keeps each locale URL canonical for itself', () => {
		expect(canonicalUrl('/it/hub', origin, 'it')).toBe('https://credimi.io/it/hub');
	});

	it('lists every locale plus x-default', () => {
		const hreflangs = localeAlternates('/hub', origin).map((alternate) => alternate.hreflang);
		expect(hreflangs).toEqual(['en', 'it', 'es-es', 'de', 'da', 'pt-br', 'fr', 'x-default']);
	});

	it('names the localized home page without the trailing slash SvelteKit redirects', () => {
		const italian = localeAlternates('/', origin).find(
			(alternate) => alternate.hreflang === 'it'
		);
		expect(italian?.href).toBe('https://credimi.io/it');
	});
});
