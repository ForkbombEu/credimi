// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getExceptionMessage } from '@/utils/errors';

//

export class BaseError extends Error {
	constructor(e: unknown) {
		super(getExceptionMessage(e));
	}
}

export class NotFoundError extends BaseError {}
