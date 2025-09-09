// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { types as t, task } from '#';
import type { Simplify } from 'type-fest';

/* Crud */

export interface DefaultConfig {
	type: object;
	input?: object;
	key: string | number;
	keyName?: string;
	options?: object;
}

export type Record<C extends DefaultConfig> = Simplify<
	C['type'] & { [K in NonNullable<C['keyName']>]: C['key'] }
>;

export type Task<Config extends DefaultConfig> = task.Task<Record<Config>, t.BaseError>;

export interface Crud<c extends DefaultConfig> {
	read(key: c['key'], options?: c['options']): Task<c>;
	readAll(options?: c['options']): task.Task<Record<c>[], t.BaseError>;
	create(input: c['input'], options?: c['options']): Task<c>;
	update(key: c['key'], input: Partial<c['input']>, options?: c['options']): Task<c>;
	delete(key: c['key'], options?: c['options']): task.Task<boolean, t.BaseError>;
}
