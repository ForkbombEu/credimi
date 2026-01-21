// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ClientResponseError } from 'pocketbase';

import { Effect as _, Either, pipe } from 'effect';
import { z, type ZodError } from 'zod/v3';

import type { SelectOption } from '@/components/ui-custom/utils';

import { pb } from '@/pocketbase';

/* Exports */

export type StandardsWithTestSuites = z.infer<typeof templateBlueprintsResponseSchema>;

export function getStandardsWithTestSuites(
	options: { fetch?: typeof fetch; forPipeline?: boolean } = {}
): Promise<StandardsWithTestSuites | Error> {
	const { fetch: fetchFn = fetch, forPipeline = false } = options;
	let url = '/api/template/blueprints';
	if (forPipeline) url += '?only_show_in_pipeline_gui=true';

	return pipe(
		_.tryPromise({
			try: () =>
				pb.send(url, {
					method: 'GET',
					fetch: fetchFn
				}),
			catch: (e) => e as ClientResponseError
		}),
		_.andThen((response) =>
			_.try({
				try: () => templateBlueprintsResponseSchema.parse(response),
				catch: (e) => e as ZodError
			})
		),
		_.either,
		_.map((e) => {
			if (Either.isLeft(e)) return e.left;
			else return e.right;
		}),
		_.runPromise
	);
}

export async function getStandardsAndVersionsFlatOptionsList(
	options = { fetch }
): Promise<SelectOption<string>[]> {
	const standards = await getStandardsWithTestSuites(options);
	if (standards instanceof Error) return [];
	return standards.flatMap((standard) =>
		standard.versions.map((version) => ({
			value: `${standard.uid}/${version.uid}`,
			label: `${standard.name} – ${version.name}`
		}))
	);
}

/* Schemas */

const standardMetadataSchema = z.object({
	uid: z.string(),
	name: z.string(),
	description: z.string(),
	standard_url: z.string(),
	latest_update: z.string(),
	external_links: z.record(z.array(z.string())).nullable(),
	disabled: z.boolean().optional()
});

const versionMetadataSchema = z.object({
	uid: z.string(),
	name: z.string(),
	latest_update: z.string(),
	specification_url: z.string().optional()
});

const suiteMetadataSchema = z.object({
	uid: z.string(),
	name: z.string(),
	homepage: z.string(),
	repository: z.string(),
	help: z.string(),
	description: z.string(),
	logo: z.string().optional()
});

const suiteSchema = suiteMetadataSchema.extend({
	files: z.array(z.string()),
	paths: z.array(z.string())
});

const versionSchema = versionMetadataSchema.extend({
	suites: z.array(suiteSchema)
});

const standardSchema = standardMetadataSchema.extend({
	versions: z.array(versionSchema)
});

const templateBlueprintsResponseSchema = z.array(standardSchema);

export type Suite = z.infer<typeof suiteSchema>;
export type Version = z.infer<typeof versionSchema>;
export type Standard = z.infer<typeof standardSchema>;
