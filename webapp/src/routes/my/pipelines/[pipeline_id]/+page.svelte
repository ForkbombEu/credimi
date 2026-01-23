<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeft } from '@lucide/svelte';
	import { PolledResource } from '$lib/utils/state.svelte.js';
	import WorkflowsTable from '$lib/workflows/workflows-table.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import { getPipelineWorkflows } from '../_partials/workflows.js';
	import { setDashboardNavbar } from '../../+layout@.svelte';

	//

	let { data } = $props();
	let { pipeline } = $derived(data);

	$effect(() => {
		setDashboardNavbar({ title: `${m.Pipelines()} | ${pipeline.name}`, right: navbarRight });
	});

	const workflows = new PolledResource(() => getPipelineWorkflows(pipeline.id), {
		initialValue: () => data.workflows
	});
</script>

{#snippet navbarRight()}
	<Button variant="outline" href="/my/pipelines">
		<ArrowLeft />
		{m.Back_to_pipelines()}
	</Button>
{/snippet}

<WorkflowsTable workflows={workflows.current ?? []} />
