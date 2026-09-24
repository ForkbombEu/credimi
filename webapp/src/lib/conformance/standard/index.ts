// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ListNestOptions } from '../client.js';

import * as Store from '../store.svelte.js';

import { resolveCheckPathFromNest, type ResolvedCheckPath } from './resolve-check-path.js';
import { resolveSuite } from './resolve-suite.js';

export { resolveSuite };
export {
	resolveCheckPathFromNest,
	type ResolvedCheckPath,
	type ResolvedNestPath
} from './resolve-check-path.js';
export * as Store from '../store.svelte.js';

/**
 * Ensure nest browse is loaded, then resolve a filesystem check path
 * (`standard/version/suite/test`). Catalog surface is required (feeds Store.load).
 */
export async function resolveCheckPath(
	checkId: string,
	options: Pick<ListNestOptions, 'surface' | 'fetch'>
): Promise<ResolvedCheckPath | null> {
	const standards = await Store.load(options);
	const resolved = resolveCheckPathFromNest(standards, checkId);
	if (!resolved?.file) return null;
	return {
		standard: resolved.standard,
		version: resolved.version,
		suite: resolved.suite,
		test: resolved.file
	};
}
