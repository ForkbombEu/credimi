// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Either, pipe, Effect as _ } from 'effect';
import type { SelectOption } from '@/components/ui-custom/utils';
import { pb } from '@/pocketbase';
import type { ClientResponseError } from 'pocketbase';
import { z, type ZodError } from 'zod';

/* Exports */

export type StandardsWithTestSuites = z.infer<typeof templateBlueprintsResponseSchema>;

export function getStandardsWithTestSuites(options = { fetch }) {
	return pipe(
		_.tryPromise({
			try: () =>
				pb.send('/api/template/blueprints', {
					method: 'GET',
					fetch: options.fetch
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
		_.runPromise
	);
}

export async function getStandardsAndVersionsFlatOptionsList(
	options = { fetch }
): Promise<SelectOption<string>[]> {
	const response = await getStandardsWithTestSuites(options);
	if (!Either.isRight(response)) return [];
	const standards = response.right;
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
	description: z.string()
});

const suiteSchema = suiteMetadataSchema.extend({
	files: z.array(z.string())
});

export type Suite = z.infer<typeof suiteSchema>;

const versionSchema = versionMetadataSchema.extend({
	suites: z.array(suiteSchema)
});

const standardSchema = standardMetadataSchema.extend({
	versions: z.array(versionSchema)
});

const templateBlueprintsResponseSchema = z.array(standardSchema);
