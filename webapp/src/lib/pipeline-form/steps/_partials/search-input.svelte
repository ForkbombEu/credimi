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
		name?: string;
	};

	let { search, placeholder = 'Search', name }: Props = $props();
</script>

<div class="relative">
	<Input bind:value={search.text} {placeholder} {name} />

	<div class="absolute top-0 right-0 p-1">
		{#if search.text}
			<IconButton icon={XIcon} variant="ghost" size="sm" onclick={() => search.clear()} />
		{:else}
			<IconButton icon={SearchIcon} variant="ghost" size="sm" disabled />
		{/if}
	</div>
</div>
