// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getExceptionMessage } from '@/utils/errors';

//

export class BaseError extends Error {
	original: unknown;
	constructor(e: unknown) {
		super(getExceptionMessage(e));
		this.original = e;
	}
}

export class NotFoundError extends BaseError {}

export type GenericRecord = Record<string, unknown>;

export type State<T> = {
	current: T;
};
