// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export function getExceptionMessage(e: unknown): string {
	if (e instanceof Error) {
		return e.message;
	} else {
		return JSON.stringify(e);
	}
}

export function exceptionToError(e: unknown): Error {
	if (e instanceof Error) {
		return e;
	} else {
		return new Error(`Unexpected error: ${JSON.stringify(e)}`);
	}
}

//

export class NotBrowserError extends Error {}
