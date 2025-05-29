// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z, ZodError } from 'zod';
import { Effect as _, pipe } from 'effect';
import type { ClientResponseError } from 'pocketbase';
import { pb } from '@/pocketbase';

//

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

export const templateBlueprintsResponseSchema = z.array(standardSchema);
export type StandardsWithTestSuites = z.infer<typeof templateBlueprintsResponseSchema>;

/** */

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
