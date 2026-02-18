<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { getEnrichedPipeline } from '$lib/pipeline-form/functions';

	import { pageDetails } from './_utils/types';

	//

	export async function getPipelineDetails(itemId: string, fetchFn = fetch) {
		const pipeline = await getEnrichedPipeline(itemId, { fetch: fetchFn });
		return pageDetails('pipelines', { pipeline });
	}
</script>

<script lang="ts">
	import StepCardDisplay from '$lib/pipeline-form/steps-builder/_partials/step-card-display.svelte';

	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as s } from './_utils/sections';

	//

	type Props = Awaited<ReturnType<typeof getPipelineDetails>>;
	let { pipeline }: Props = $props();
</script>

<LayoutWithToc sections={[s.description, s.pipeline_steps, s.workflow_yaml]}>
	<DescriptionSection description={pipeline.record.description} />

	<PageSection indexItem={s.pipeline_steps} empty={pipeline.steps.length === 0}>
		<div class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
			{#each pipeline.steps as step, index (index)}
				<StepCardDisplay {step} readonly showLinkToMarketplace>
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
