<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { toWorkflowStatusReadable } from '@forkbombeu/temporal-ui';
	import { ArrowLeft } from '@lucide/svelte';
	import { Pipeline } from '$lib';
	import { PolledResource } from '$lib/utils/state.svelte.js';

	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator/index.js';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../../+layout@.svelte';

	//

	let { data } = $props();
	let { pipeline } = $derived(data);

	$effect(() => {
		setDashboardNavbar({ title: m.Pipelines(), right: navbarRight });
	});

	const workflows = new PolledResource(() => Pipeline.Workflows.list(pipeline.id), {
		initialValue: () => data.workflows
	});

	//

	// Filter out canceled runs for success rate calculation
	const nonCanceledWorkflows = $derived(
		workflows.current?.filter((w) => {
			const status = toWorkflowStatusReadable(w.status);
			return status !== 'Canceled';
		}) ?? []
	);

	const totalRuns = $derived(nonCanceledWorkflows.length);

	const totalSuccesses = $derived(
		nonCanceledWorkflows.filter((w) => {
			const status = toWorkflowStatusReadable(w.status);
			return status === 'Completed';
		}).length
	);

	const successRate = $derived(((totalSuccesses / totalRuns) * 100).toFixed(1) + '%');
</script>

{#snippet navbarRight()}
	<Button variant="outline" href="/my/pipelines">
		<ArrowLeft />
		{m.Back_to_pipelines()}
	</Button>
{/snippet}

<div class="flex justify-between gap-8">
	<div class="-space-y-1">
		<T class="text-muted-foreground">{m.Pipeline()}</T>
		<T tag="h2">{pipeline.name}</T>
	</div>

	<div class="flex flex-wrap gap-2 md:flex-nowrap">
		{@render numberBox(totalRuns, m.Total_runs())}
		{@render numberBox(totalSuccesses, m.Successful_runs())}
		{@render numberBox(successRate, m.Success_rate())}
	</div>
</div>

<Separator />

<T tag="h3">{m.workflow_runs()}</T>
<Pipeline.Workflows.Table workflows={workflows.current ?? []} />

<!--  -->

{#snippet numberBox(highlight: string | number, description: string)}
	<div class="flex h-20 w-[140px] flex-col items-start justify-between rounded-lg border p-3">
		<T tag="h2" class="mb-0! pb-0!">{highlight}</T>
		<T class="text-sm">{description}</T>
	</div>
{/snippet}
