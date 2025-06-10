// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getLocale } from '@/i18n';

export const contentLoaders = import.meta.glob('$lib/content/**/*.md', { query: '?raw' });

export function getContentBySlug(slug: string): string | undefined {
	const locale = getLocale();
	//
}
