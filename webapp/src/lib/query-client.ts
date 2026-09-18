// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { QueryClient } from '@tanstack/svelte-query';
import { browser } from '$app/environment';

/**
 * App-wide QueryClient. Prefer `createQuery` inside components (context from
 * QueryClientProvider). Pass this client explicitly when creating queries
 * outside component init — e.g. pipeline step forms built in `$effect.root()`.
 */
export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			enabled: browser
		}
	}
});
