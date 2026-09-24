// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ListNestOptions } from './client';
import type { Standard } from './types';

import { listNest } from './client';

//

const standards = $state<Standard[]>([]);

/** Surface key last successfully loaded into {@link standards}. */
let loadedSurface: string | undefined;
/** In-flight load for dedupe (hooks + deserialize share one fetch). */
let inflight: Promise<readonly Standard[]> | undefined;
let inflightSurface: string | undefined;

const readonlyView = {
	get standards(): readonly Standard[] {
		return standards as readonly Standard[];
	}
};

export function get() {
	return readonlyView;
}

/**
 * Sole client Filesystem-axis nest projection (ADR-0004). Awaitable nest browse
 * load — idempotent per surface; concurrent callers share one in-flight fetch.
 * Rejects on catalog errors (no fire-and-forget). Client callers must load via
 * this Store; SSR / non-hydrating reads use getNestStandards.
 * Catalog surface is required (no silent default).
 */
export async function load(
	options: Pick<ListNestOptions, 'surface' | 'fetch'>
): Promise<readonly Standard[]> {
	const key = options.surface;
	if (loadedSurface === key) return get().standards;
	if (inflight && inflightSurface === key) return inflight;

	inflightSurface = key;
	inflight = (async () => {
		const result = await listNest(options);
		if (result.isErr) throw result.error;
		standards.length = 0;
		standards.push(...result.value);
		loadedSurface = key;
		return get().standards;
	})().finally(() => {
		inflight = undefined;
		inflightSurface = undefined;
	});

	return inflight;
}
