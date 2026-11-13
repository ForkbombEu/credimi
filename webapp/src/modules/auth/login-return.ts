// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createSessionStorageHandlers } from '@/utils/storage';

//

type LoginReturnSession = { path: string };

/** The private page a signed-out visitor asked for, opened again after login. */
export const LoginReturnSession =
	createSessionStorageHandlers<LoginReturnSession>('login_return_to');

const ORIGIN = 'https://credimi.invalid';

/**
 * Returns `path` when it is a same-origin path under `/my`, the only pages a
 * login returns to, and undefined otherwise.
 */
export function loginReturnPath(path: string): string | undefined {
	if (!path.startsWith('/') || path.startsWith('//') || path.includes('\\')) return undefined;
	let url: URL;
	try {
		url = new URL(path, ORIGIN);
	} catch {
		return undefined;
	}
	if (url.origin !== ORIGIN) return undefined;
	if (url.pathname !== '/my' && !url.pathname.startsWith('/my/')) return undefined;
	return url.pathname + url.search;
}
