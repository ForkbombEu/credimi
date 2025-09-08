// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getContext as svelteGetContext, setContext as svelteSetContext } from 'svelte';

//

export function createContextHandlers<Context>(key: string) {
	const contextKey = Symbol(key);

	function setContext<C extends Context>(context: C) {
		return svelteSetContext<Context>(contextKey, context);
	}

	function getContext<C extends Context>(): C {
		return svelteGetContext<C>(contextKey);
	}

	return {
		setContext,
		getContext,
		contextKey
	};
}
