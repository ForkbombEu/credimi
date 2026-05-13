// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/Task';
import { z, ZodError } from 'zod';

import { pb } from '@/pocketbase';

import { standardSchema } from './types';

//

const listAllResponseSchema = z.array(standardSchema);

export type ListAllResponse = z.infer<typeof listAllResponseSchema>;

export function listAll(
	options: { fetch?: typeof fetch; forPipeline?: boolean } = {}
): Task.Task<ListAllResponse, ClientResponseError | ZodError> {
	const { fetch: fetchFn = fetch, forPipeline = false } = options;

	let path = '/api/template/blueprints';
	if (forPipeline) path += '?only_show_in_pipeline_gui=true';

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => pb.send(path, { method: 'GET', fetch: fetchFn })
	).andThen((response) => {
		const res = listAllResponseSchema.safeParse(response);
		if (res.success) return Task.resolve(res.data);
		else return Task.reject(res.error);
	});
}
