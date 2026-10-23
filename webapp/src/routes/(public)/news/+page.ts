// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { staticPages } from '$lib/seo';

export const load = () => ({ seo: staticPages['/news'] });
