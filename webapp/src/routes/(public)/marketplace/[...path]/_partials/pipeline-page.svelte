<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { fetchPipeline } from '$lib/pipeline-form/serde';

	import { pb } from '@/pocketbase';

	import { pageDetails } from './_utils/types';

	export async function getPipelineDetails(itemId: string, fetchFn = fetch) {
		const pipelineData = await fetchPipeline(itemId, { fetch: fetchFn });

		return pageDetails('pipelines', {
			pipelineData,
			yaml: pipelineData.yaml,
			description: pipelineData.description
		});
	}
</script>

<script lang="ts">
	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as s } from './_utils/sections';
	import PipelineStepsDisplay from './pipeline-steps-display.svelte';

	type Props = Awaited<ReturnType<typeof getPipelineDetails>>;
	let { pipelineData, yaml, description }: Props = $props();
</script>

<LayoutWithToc sections={[s.description, s.pipeline_steps, s.workflow_yaml]}>
	<DescriptionSection {description} />

	<PageSection indexItem={s.pipeline_steps} empty={pipelineData.steps.length === 0}>
		<PipelineStepsDisplay steps={pipelineData.steps} />
	</PageSection>

	<CodeSection indexItem={s.workflow_yaml} code={yaml} language="yaml" />
</LayoutWithToc>
