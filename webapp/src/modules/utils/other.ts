// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { dev } from '$app/environment';
import { deLocalizeUrl } from '@/i18n';
import type { Page } from '@sveltejs/kit';

//

export function getJsonDataSize(data: unknown): number {
	return new Blob([JSON.stringify(data)]).size;
}

export function capitalize(text: string) {
	return text.charAt(0).toUpperCase() + text.slice(1);
}

export function createDummyFile(options: { filename?: string; size?: number; mime?: string } = {}) {
	const { filename = 'file.txt', size = 10, mime = 'text/plain' } = options;
	return new File(['a'.repeat(size)], filename, {
		type: mime,
		lastModified: Date.now()
	});
}

//

export function isLinkActive(href: string, page: Page, includeSubpages = false) {
	const isExact = page.url.pathname == deLocalizeUrl(href).pathname;
	const isParent = page.url.pathname.includes(deLocalizeUrl(href).pathname);
	return includeSubpages ? isParent : isExact;
}

//

export function removeTrailingSlash(text: string) {
	if (text.endsWith('/')) {
		return text.slice(0, -1);
	}
	return text;
}

//

export function wait(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

//

export type MaybeArray<T> = T | T[] | undefined | null;

export function ensureArray<T>(data: MaybeArray<T>): T[] {
	if (Array.isArray(data)) return data;
	if (data) return [data];
	else return [];
}

export function maybeArrayIsValue<T>(data: MaybeArray<T>): boolean {
	if (Array.isArray(data)) return Boolean(data.length);
	else return Boolean(data);
}

//

export function log(...data: unknown[]) {
	if (dev) console.log(...data);
}

export function warn(...data: unknown[]) {
	if (dev) console.warn(...data);
}

export function pipeLog<T>(data: T): T {
	log(data);
	return data;
}
