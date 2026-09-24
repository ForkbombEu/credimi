<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Debounced } from 'runed';

	import SearchInput from '@/components/ui-custom/search-input.svelte';
	import { m } from '@/i18n';

	import { getCollectionManagerContext } from './collectionManagerContext';

	//

	type Props = {
		class?: string;
		containerClass?: string;
		placeholder?: string;
	};

	let { class: className, containerClass = '', placeholder = m.Search() }: Props = $props();

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

<SearchInput
	bind:value={searchText}
	{placeholder}
	class={containerClass}
	inputClass={className}
	onclear={() => {
		manager.query.clearSearch();
		searchText = '';
	}}
/>
