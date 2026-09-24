// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ListAllOptions } from '../client.js';

import * as Store from '../store.svelte.js';

import { resolveCheckPathFromNest, type ResolvedCheckPath } from './resolve-check-path.js';
import { resolveSuite } from './resolve-suite.js';

export { resolveSuite };
export { resolveCheckPathFromNest, type ResolvedCheckPath } from './resolve-check-path.js';
export * as Store from '../store.svelte.js';

/**
 * Ensure nest browse is loaded, then resolve a filesystem check path
 * (`standard/version/suite/test`). Catalog surface is required (feeds Store.load).
 */
export async function resolveCheckPath(
	checkId: string,
	options: Pick<ListAllOptions, 'surface' | 'fetch'>
): Promise<ResolvedCheckPath | null> {
	const standards = await Store.load(options);
	return resolveCheckPathFromNest(standards, checkId);
}
