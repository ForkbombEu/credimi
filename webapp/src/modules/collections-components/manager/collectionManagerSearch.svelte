<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { String } from 'effect';
	import { X } from 'lucide-svelte';
	import { Debounced } from 'runed';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { Input } from '@/components/ui/input';
	import { m } from '@/i18n';

	import { getCollectionManagerContext } from './collectionManagerContext';

	//

	type Props = {
		class?: string;
		containerClass?: string;
	};

	let { class: className, containerClass = '' }: Props = $props();

	const { manager } = $derived(getCollectionManagerContext());

	let searchText = $state('');
	const deboucedSearch = new Debounced(() => searchText, 500);

	$effect(() => {
		manager.query.setSearch(deboucedSearch.current);
	});

	$effect(() => {
		if (!manager.query.hasSearch()) {
			searchText = '';
		}
	});
</script>

<div class="relative flex {containerClass}">
	<Input bind:value={searchText} placeholder={m.Search()} class={className} />
	{#if String.isString(searchText)}
		<Button
			onclick={() => {
				manager.query.clearSearch();
				searchText = '';
			}}
			class="absolute right-1 top-1 size-8"
			variant="ghost"
		>
			<Icon src={X} size="" />
		</Button>
	{/if}
</div>
