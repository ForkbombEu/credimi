<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import ItemCard from '$lib/pipeline-form/steps/_partials/item-card.svelte';
	import SearchInput from '$lib/pipeline-form/steps/_partials/search-input.svelte';
	import WithEmptyState from '$lib/pipeline-form/steps/_partials/with-empty-state.svelte';
	import WithLabel from '$lib/pipeline-form/steps/_partials/with-label.svelte';

	import type { SelectRunnerForm } from './select-runner.svelte.js';

	//

	type Props = {
		form: SelectRunnerForm;
		showSelected?: boolean;
	};

	let { form, showSelected = true }: Props = $props();
</script>

{#if showSelected && form.selectedRunner}
	<div class="flex flex-col gap-4 border-b p-4">
		<WithLabel label={m.Runner()}>
			<ItemCard title={form.selectedRunner.name} onDiscard={() => form.removeRunner()} />
		</WithLabel>
	</div>
{/if}

{#if !form.selectedRunner}
	<WithLabel label={m.Runner()} class="p-4">
		<SearchInput search={form.runnerSearch} />
	</WithLabel>

	<WithEmptyState items={form.foundRunners} emptyText={m.No_runners_found()}>
		{#snippet item({ item })}
			<ItemCard title={item.name} onClick={() => form.selectRunner(item)}>
				{#snippet right()}
					{#if !item.published}
						<Badge variant="secondary">
							{m.private()}
						</Badge>
					{/if}
				{/snippet}
			</ItemCard>
		{/snippet}
	</WithEmptyState>
{/if}
