<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import SectionTitle from '@/components/ui-custom/sectionTitle.svelte';
	import RecordCreate from './record-actions/recordCreate.svelte';
	import { getCollectionManagerContext } from './collectionManagerContext';
	import type { Snippet } from 'svelte';

	interface Props {
		title?: string | undefined;
		hideCreate?: boolean;
		right?: Snippet;
		buttonContent?: Snippet;
	}

	const { title, hideCreate = false, right: rightSnippet, buttonContent }: Props = $props();
	const { manager } = $derived(getCollectionManagerContext());
</script>

<SectionTitle title={title ?? manager.collection}>
	{#snippet right()}
		{#if rightSnippet}
			{@render rightSnippet()}
		{:else if !hideCreate}
			<RecordCreate buttonText={buttonContent} />
		{/if}
	{/snippet}
</SectionTitle>
