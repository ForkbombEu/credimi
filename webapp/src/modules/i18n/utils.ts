// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Handle, Page } from '@sveltejs/kit';

import { redirect as svelteKitRedirect } from '@sveltejs/kit';
import { goto as svelteKitGoto } from '$app/navigation';
import { resolve } from '$app/paths';
import { Record } from 'effect';

import { getLocale, localizeHref, localizeUrl, type Locale } from './paraglide/runtime';
import { paraglideMiddleware } from './paraglide/server';

//

export const handleParaglide: Handle = ({ event, resolve }) =>
	paraglideMiddleware(event.request, ({ request, locale }) => {
		event.request = request;

		return resolve(event, {
			transformPageChunk: ({ html }) => html.replace('%paraglide.lang%', locale)
		});
	});

export const goto = (url: string) => svelteKitGoto(resolve(localizeHref(url) as '/')); // TS escape hatch

export const redirect = (url: string) => svelteKitRedirect(303, localizeUrl(url));

//

export const languagesDisplay: Record<Locale, { flag: string; name: string }> = {
	en: { flag: '🇬🇧', name: 'English' },
	it: { flag: '🇮🇹', name: 'Italiano' },
	de: { flag: '🇩🇪', name: 'Deutsch' },
	fr: { flag: '🇫🇷', name: 'Français' },
	da: { flag: '🇩🇰', name: 'Dansk' },
	'pt-br': { flag: '🇧🇷', name: 'Português' },
	'es-es': { flag: '🇪🇸', name: 'Español' }
};

export function getLanguagesData(page: Page): LanguageData[] {
	const currentLang = getLocale();

	return Record.keys(languagesDisplay).map((lang) => ({
		tag: lang,
		href: localizeHref(page.url.pathname, { locale: lang }),
		hreflang: lang,
		flag: languagesDisplay[lang].flag,
		name: languagesDisplay[lang].name,
		isCurrent: lang == currentLang
	}));
}

export type LanguageData = {
	tag: Locale;
	href: string;
	hreflang: Locale;
	flag: string;
	name: string;
	isCurrent: boolean;
};
