// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Reroute } from '@sveltejs/kit';

import { deLocalizeUrl } from '@/i18n';

export const reroute: Reroute = (request) => {
	return deLocalizeUrl(request.url).pathname;
};
