<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';

	import T from '@/components/ui-custom/t.svelte';

	import type { PageData } from '../+page';

	import PageSection from '../../../[...path]/_partials/_utils/page-section.svelte';
	import { sections as s } from '../../../[...path]/_partials/_utils/sections';
	import CheckCard from './components/check-card.svelte';
	import PageLayout from './components/page-layout.svelte';

	//

	type Props = Extract<PageData, { type: 'collection-page' }>;

	let { standard, version, suite }: Props = $props();

	//

	const tocSections: IndexItem[] = [s.description, s.checks];
</script>

<PageLayout {tocSections}>
	{#snippet top()}
		<T tag="h1">
			{standard.name} • {version.name}
		</T>
		<T tag="h3">{suite.name}</T>
	{/snippet}

	{#snippet content()}
		<PageSection indexItem={s.description}>
			<p>{standard.description}</p>
			<p>{suite.description}</p>
		</PageSection>

		<PageSection indexItem={s.checks}>
			<div class="space-y-2">
				{#each suite.paths as path (path)}
					<CheckCard {standard} {version} {suite} test={path} />
				{/each}
			</div>
		</PageSection>
	{/snippet}
</PageLayout>
