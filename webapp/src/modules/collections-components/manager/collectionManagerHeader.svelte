<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import SectionTitle from '@/components/ui-custom/sectionTitle.svelte';

	import { getCollectionManagerContext } from './collectionManagerContext';
	import RecordCreate from './record-actions/recordCreate.svelte';

	interface Props {
		title?: string | undefined;
		hideCreate?: boolean;
		right?: Snippet;
		buttonContent?: Snippet;
		id?: string;
	}

	const { title, hideCreate = false, right: rightSnippet, buttonContent, id }: Props = $props();
	const { manager } = $derived(getCollectionManagerContext());
</script>

<SectionTitle title={title ?? manager.collection} {id}>
	{#snippet right()}
		{#if rightSnippet}
			{@render rightSnippet()}
		{:else if !hideCreate}
			<RecordCreate buttonText={buttonContent} />
		{/if}
	{/snippet}
</SectionTitle>
