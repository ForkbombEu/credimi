// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Merge } from 'type-fest';

import { error } from '@sveltejs/kit';
import { Hub } from '$lib';
import { getNestStandards } from '$lib/conformance';
import { resolveCheckPathFromNest } from '$lib/conformance/standard/resolve-check-path';
import { ogImages, plainText } from '$lib/seo';

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
	const suiteTitle = `${suite.name} ${version.name}`;
	const baseData = {
		standard,
		version,
		suite,
		basePath: Hub.Conformance.getSuitePageUrl(standard.uid, version.uid, suite.uid)
	};

	if (!file) {
		return pageDetails('collection-page', {
			...baseData,
			seo: {
				title: `${suiteTitle} conformance checks for ${standard.name}`,
				description:
					plainText(suite.description) ||
					`${suite.members.length} ${standard.name} conformance checks from ${suite.name}, runnable in Credimi pipelines against real wallets, issuers and verifiers.`,
				image: ogImages.conformanceAutomation
			}
		});
	}
	// Check titles are full test descriptions; the check id is the short, searchable name.
	const checkDescription = suite.members.find((m) => m.path.endsWith(`/${file}`))?.title;
	return pageDetails('file-page', {
		...baseData,
		file,
		seo: {
			title: `${file}: ${standard.name} ${version.name} conformance check`,
			description: plainText(
				`${checkDescription ? `${checkDescription} ` : ''}A ${standard.name} ${version.name} conformance check from ${suite.name}, runnable in a Credimi pipeline.`
			),
			image: ogImages.conformanceAutomation
		}
	});
};

function pageDetails<K extends string, Data extends object>(
	type: K,
	data: Data
): Merge<{ type: K }, Data> {
	return { type, ...data };
}

export type PageData = Awaited<ReturnType<typeof load>>;
