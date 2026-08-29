// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { FCAF_TESTS, type FCAFTestCatalogEntry } from '$lib/fcaf/tests.generated.js';

//

export type FCAFGroupedTests = {
	key: string;
	label: string;
	category: string;
	tests: FCAFTestCatalogEntry[];
};

export function groupLabel(section: string): string {
	return humanize(section.split('.').pop() ?? section);
}

export function categoryLabel(section: string): string {
	return humanize(section.split('.')[0] ?? section);
}

export function groupSelectedTests(testIds: string[]): FCAFGroupedTests[] {
	const acc: Record<string, FCAFTestCatalogEntry[]> = {};
	for (const test of FCAF_TESTS) {
		if (!testIds.includes(test.id)) continue;
		const key = test.section || 'other';
		(acc[key] ??= []).push(test);
	}

	return Object.entries(acc)
		.sort(([a], [b]) => a.localeCompare(b))
		.map(([key, tests]) => ({
			key,
			label: groupLabel(key),
			category: categoryLabel(key),
			tests
		}));
}

function humanize(key: string): string {
	const word = key.replace(/_/g, ' ');
	return word.charAt(0).toUpperCase() + word.slice(1);
}
