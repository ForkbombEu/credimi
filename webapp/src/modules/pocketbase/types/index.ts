// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export * from './extra.generated';
export * from './index.generated';

//

import type { SimplifyDeep } from 'type-fest';

export const systemFields = [
	// base system fields
	'id',
	'created',
	'updated',
	// user/auth fields
	'password',
	'tokenKey',
	'username'
] as const;

export type Data<R extends Record<string, unknown>> = SimplifyDeep<
	Omit<R, (typeof systemFields)[number]>
>;

export type { CollectionName } from '@/pocketbase/collections-models';
