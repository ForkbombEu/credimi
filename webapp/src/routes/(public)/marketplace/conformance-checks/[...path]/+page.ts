// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Merge } from 'type-fest';

import { error } from '@sveltejs/kit';

//

export const load = async ({ params, parent }) => {
	const { path } = params;
	const { conformanceChecks } = await parent();

	const chunks = path.split('/');
	if (chunks.length < 3) error(404);

	const [standardUid, versionUid, suiteUid] = chunks;
	const file = chunks.at(3);

	const standard = conformanceChecks.find((standard) => standard.uid === standardUid);
	const version = standard?.versions.find((version) => version.uid === versionUid);
	const suite = version?.suites.find((suite) => suite.uid === suiteUid);

	if (!standard || !version || !suite) error(404);

	const baseData = {
		standard,
		version,
		suite,
		basePath: `/marketplace/conformance-checks/${standardUid}/${versionUid}/${suiteUid}`
	};

	if (!file) {
		return pageDetails('collection-page', baseData);
	} else {
		return pageDetails('file-page', {
			...baseData,
			file
		});
	}
};

function pageDetails<K extends string, Data extends object>(
	type: K,
	data: Data
): Merge<{ type: K }, Data> {
	return { type, ...data };
}

export type PageData = Awaited<ReturnType<typeof load>>;
