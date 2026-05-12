<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { getEnrichedPipeline } from '$lib/pipeline-form/functions';

	import type { PipelineScoreboardCacheResponse } from '@/pocketbase/types';

	import { pageDetails } from './_utils/types';

	//

	export async function getPipelineDetails(itemId: string, fetchFn = fetch) {
		const pipeline = await getEnrichedPipeline(itemId, { fetch: fetchFn });

		let results: PipelineScoreboardCacheResponse | undefined;
		try {
			results = await pb
				.collection('pipeline_scoreboard_cache')
				.getFirstListItem(
					pb.filter('pipeline = {:pipeline}', { pipeline: pipeline.record.id })
				);
		} catch (e) {
			console.error(e);
		}

		return pageDetails('pipelines', { pipeline, results });
	}
</script>

<script lang="ts">
	import { TableIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import StepCardDisplay from '$lib/pipeline-form/steps-builder/_partials/step-card-display.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as s } from './_utils/sections';

	//

	type Props = Awaited<ReturnType<typeof getPipelineDetails>>;
	let { pipeline, results }: Props = $props();
</script>

<LayoutWithToc sections={[s.description, s.pipeline_steps, s.workflow_yaml]}>
	<div class="flex flex-col gap-4 md:flex-row">
		<DescriptionSection description={pipeline.record.description} class="grow" />

		<PageSection indexItem={s.results} empty={!results}>
			<div class="grid-pattern space-y-4 rounded-lg border border-primary bg-background p-4">
				<div class="stat">
					<p>{m.scoreboard_success_rate()}</p>
					<p>
						{results?.total_successes} / {results?.total_runs} ({results?.success_rate}%)
					</p>
				</div>
				<div class="stat">
					<p>{m.Execution_mode()}</p>
					<p class="leading-[1.2]">
						{results?.manually_executed_runs}
						{m.executions_manual()}
						<br />
						{results?.scheduled_runs}
						{m.executions_scheduled()}
					</p>
				</div>
				<div class="stat">
					<p>{m.Min_running_time()}</p>
					<p>{results?.minimum_running_time}</p>
				</div>
			</div>

			{#snippet right()}
				<div class="pb-1 pl-6">
					<Button variant="default" size="sm" href={resolve('/scoreboard')}>
						<TableIcon />
						{m.View_Scoreboard()}
					</Button>
				</div>
			{/snippet}
		</PageSection>
	</div>

	<PageSection indexItem={s.pipeline_steps} empty={pipeline.steps.length === 0}>
		<div class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
			{#each pipeline.steps as step, index (index)}
				<StepCardDisplay {step} readonly>
					{#snippet topRight()}
						<div class="pr-2 text-xs text-muted-foreground">
							#{index + 1}
						</div>
					{/snippet}
				</StepCardDisplay>
			{/each}
		</div>
	</PageSection>

	<CodeSection indexItem={s.workflow_yaml} code={pipeline.record.yaml} language="yaml" />
</LayoutWithToc>

<style lang="postcss">
	@reference "../../../../layout.css";

	.stat {
		p:first-child {
			@apply text-sm text-muted-foreground;
		}
		p:last-child {
			@apply text-lg font-medium;
		}
	}

	.stat:first-child {
		p:last-child {
			@apply text-2xl font-bold;
		}
	}

	.grid-pattern {
		--color: color-mix(in srgb, var(--color-primary) 3%, transparent);
		--size-large: 100px;
		--size-small: 20px;
		--offset-small: 20px;
		--offset-large: -20px;
		background-color: var(--color-background);
		background-image:
			linear-gradient(var(--color) 2px, transparent 2px),
			linear-gradient(90deg, var(--color) 2px, transparent 2px),
			linear-gradient(var(--color) 1px, transparent 1px),
			linear-gradient(90deg, var(--color) 1px, var(--color-background) 1px);
		background-size:
			var(--size-large) var(--size-large),
			var(--size-large) var(--size-large),
			var(--size-small) var(--size-small),
			var(--size-small) var(--size-small);
		background-position:
			var(--offset-large) var(--offset-large),
			var(--offset-small) var(--offset-small),
			var(--offset-large) var(--offset-large),
			var(--offset-small) var(--offset-small);
	}
</style>
