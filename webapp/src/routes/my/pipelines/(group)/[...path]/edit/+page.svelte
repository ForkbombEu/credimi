<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Render } from '$lib/renderable';
	import { minimalEnrichedPipeline } from '$lib/pipeline-form/functions.js';
	import { PipelineForm } from '$pipeline-form/pipeline-form.svelte.js';

	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import { m } from '@/i18n';

	import type { EditPipelineLoad } from './+page.js';

	//

	const { data } = $props();

	function createEditForm(edit: EditPipelineLoad) {
		return new PipelineForm({
			mode: 'edit',
			organizationId: data.organization.id,
			pipeline: edit.pipeline,
			startLockedManual: edit.startLockedManual
		});
	}
</script>

{#await data.edit}
	{@const shell = createEditForm({ pipeline: minimalEnrichedPipeline(data.pipeline) })}
	<Render item={shell} />
	<LoadingDialog loading={true}>
		{m.Loading_pipeline()}
	</LoadingDialog>
{:then edit}
	{@const form = createEditForm(edit)}
	<Render item={form} />
{/await}
