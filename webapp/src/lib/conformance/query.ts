// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { z, ZodError } from 'zod';

import { pb } from '@/pocketbase';

import { standardSchema } from './types';

//

const listAllResponseSchema = z.array(standardSchema);

export type ListAllResponse = z.infer<typeof listAllResponseSchema>;
export type TemplateSurface = 'manual' | 'pipeline';

export function listAll(
	options: { fetch?: typeof fetch; surface?: TemplateSurface } = {}
): Task.Task<ListAllResponse, ClientResponseError | ZodError> {
	const { fetch: fetchFn = fetch, surface = 'manual' } = options;

	const path = `/api/template/blueprints?surface=${surface}`;

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => pb.send(path, { method: 'GET', fetch: fetchFn })
	).andThen((response) => {
		const res = listAllResponseSchema.safeParse(response);
		if (res.success) return Task.resolve(res.data);
		else return Task.reject(res.error);
	});
}
