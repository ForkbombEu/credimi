// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { deLocalizeUrl } from '@/i18n/paraglide/runtime';

export const reroute = (request) => deLocalizeUrl(request.url).pathname;
