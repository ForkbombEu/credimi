<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { enrichPipeline } from '$lib/pipeline-form/functions';

	import { pb } from '@/pocketbase/index.js';

	import { pageDetails } from './_utils/types';

	//

	export async function getPipelineDetails(itemId: string, fetchFn = fetch) {
		const pipelineRecord = await pb.collection('pipelines').getOne(itemId, { fetch: fetchFn });
		const pipeline = await enrichPipeline(pipelineRecord);

		return pageDetails('pipelines', {
			yaml: pipelineRecord.yaml,
			description: pipelineRecord.description,
			pipeline
		});
	}
</script>

<script lang="ts">
	import Alert from '@/components/ui-custom/alert.svelte';

	import CodeSection from './_utils/code-section.svelte';
	import DescriptionSection from './_utils/description-section.svelte';
	import LayoutWithToc from './_utils/layout-with-toc.svelte';
	import PageSection from './_utils/page-section.svelte';
	import { sections as s } from './_utils/sections';

	type Props = Awaited<ReturnType<typeof getPipelineDetails>>;
	let { yaml, description, pipeline }: Props = $props();
</script>

<LayoutWithToc sections={[s.description, s.pipeline_steps, s.workflow_yaml]}>
	<DescriptionSection {description} />

	<PageSection indexItem={s.pipeline_steps} empty={pipeline.steps.length === 0}>
		<Alert variant="warning">Displaying steps is not implemented yet</Alert>
		<!-- <PipelineStepsDisplay steps={pipeline.steps} /> -->
	</PageSection>

	<CodeSection indexItem={s.workflow_yaml} code={yaml} language="yaml" />
</LayoutWithToc>
