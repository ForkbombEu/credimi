// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { templateBlueprintsResponseSchema } from './_partials/standards-response-schema.js';
import { error } from '@sveltejs/kit';
import { Effect as _, Either } from 'effect';
import { pipe } from 'effect';
import type { ClientResponseError } from 'pocketbase';
import type { ZodError } from 'zod';

//

export const load = async ({ fetch }) => {
	const result = await pipe(
		_.tryPromise({
			try: () =>
				pb.send('/api/template/blueprints', {
					method: 'GET',
					fetch
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

	if (Either.isLeft(result)) {
		error(500, { message: result.left.message });
	} else {
		return {
			standardsAndTestSuites: result.right
		};
	}
};
