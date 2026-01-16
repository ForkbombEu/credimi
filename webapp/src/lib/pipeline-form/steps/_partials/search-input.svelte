<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { SearchIcon, XIcon } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { Input } from '@/components/ui/input/index.js';

	import { Search } from './search.svelte.js';

	//

	type Props = {
		search: Search;
		placeholder?: string;
	};

	let { search, placeholder = 'Search' }: Props = $props();
</script>

<div class="relative">
	<Input bind:value={search.text} {placeholder} />

	<div class="absolute right-0 top-0 p-1">
		{#if search.text}
			<IconButton icon={XIcon} variant="ghost" size="sm" onclick={() => search.clear()} />
		{:else}
			<IconButton icon={SearchIcon} variant="ghost" size="sm" disabled />
		{/if}
	</div>
</div>
