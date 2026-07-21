// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { m } from '@/i18n';

export function getExceptionMessage(e: unknown): string {
	if (e instanceof Error) {
		return e.message;
	} else {
		return JSON.stringify(e);
	}
}

export function exceptionToError(e: unknown): Error {
	if (e instanceof Error) {
		return e;
	} else {
		return new Error(`Unexpected error: ${JSON.stringify(e)}`);
	}
}

export function localizePocketBaseErrorCode(
	code: string | undefined,
	fallback: string,
	catalog: Record<string, unknown> = m as unknown as Record<string, unknown>
): string {
	if (!code) return fallback;

	const candidate = catalog[code];
	if (typeof candidate !== 'function') return fallback;

	try {
		const localized = (candidate as () => unknown)();
		return typeof localized === 'string' ? localized : fallback;
	} catch {
		return fallback;
	}
}

//

export class NotBrowserError extends Error {}
