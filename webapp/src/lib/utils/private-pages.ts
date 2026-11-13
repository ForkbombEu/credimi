// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { deLocalizeUrl } from '@/i18n/paraglide/runtime';

/** Headers that keep a page out of frames on other sites (clickjacking). */
export const noFramingHeaders = {
	'content-security-policy': "frame-ancestors 'none'",
	'x-frame-options': 'DENY'
} as const;

/** Whether url is a signed-in page under `/my`, in any locale (`/my`, `/my/...`, `/it/my/...`). */
export function isPrivatePage(url: URL): boolean {
	const { pathname } = deLocalizeUrl(url);
	return pathname === '/my' || pathname.startsWith('/my/');
}
