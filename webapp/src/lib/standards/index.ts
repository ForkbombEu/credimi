// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Flat standard/version option lists for forms (verifiers, etc.).
 * Backed by PocketBase `conformance_checks` via `$lib/conformance`.
 */

import type { SelectOption } from '@/components/ui-custom/utils';

import { listAll, type TemplateSurface } from '$lib/conformance';

export type { TemplateSurface };

export async function getStandardsAndVersionsFlatOptionsList(
	options: { fetch?: typeof fetch; surface?: TemplateSurface } = {}
): Promise<SelectOption<string>[]> {
	const { fetch: fetchFn = fetch, surface = 'manual' } = options;
	const result = await listAll({ fetch: fetchFn, surface });
	if (result.isErr) return [];
	return result.value.flatMap((standard) =>
		standard.versions.map((version) => ({
			value: `${standard.uid}/${version.uid}`,
			label: `${standard.name} – ${version.name}`
		}))
	);
}
