<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import type { Header, Table } from '@tanstack/table-core';
	import type { ScoreboardRow } from '$lib/scoreboard-v2/types';

	import { createContext } from 'svelte';

	type HeaderContext = {
		header: Header<ScoreboardRow, unknown>;
		table: Table<ScoreboardRow>;
	};

	export const [getHeaderContext, setHeaderContext] = createContext<HeaderContext>();
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = HeaderContext & {
		children: Snippet;
	};

	let { children, header, table }: Props = $props();

	const contextValue: HeaderContext = {
		get header() {
			return header;
		},
		get table() {
			return table;
		}
	};

	setHeaderContext(contextValue);
</script>

{@render children?.()}
