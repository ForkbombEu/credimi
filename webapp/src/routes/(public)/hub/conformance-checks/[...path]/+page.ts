// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Merge } from 'type-fest';

import { error } from '@sveltejs/kit';
import { getNestStandards } from '$lib/conformance';
import { resolveCheckPathFromNest } from '$lib/conformance/standard/resolve-check-path';
import { Hub } from '$lib';

//

export const load = async ({ params, fetch }) => {
	const { path } = params;

	const conformanceChecks = await getNestStandards({ fetch, surface: 'pipeline' });
	if (conformanceChecks instanceof Error) {
		error(500, { message: conformanceChecks.message });
	}

	const resolved = resolveCheckPathFromNest(conformanceChecks, path);
	if (!resolved) error(404);

	const { standard, version, suite, file } = resolved;
	const baseData = {
		standard,
		version,
		suite,
		basePath: Hub.Conformance.getSuitePageUrl(standard.uid, version.uid, suite.uid)
	};

	if (!file) {
		return pageDetails('collection-page', baseData);
	}
	return pageDetails('file-page', {
		...baseData,
		file
	});
};

function pageDetails<K extends string, Data extends object>(
	type: K,
	data: Data
): Merge<{ type: K }, Data> {
	return { type, ...data };
}

export type PageData = Awaited<ReturnType<typeof load>>;
