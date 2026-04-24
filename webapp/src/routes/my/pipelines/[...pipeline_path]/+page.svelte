<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import BackButton from '$lib/layout/back-button.svelte';
	import { PolledResource } from '$lib/utils/state.svelte.js';
	import { queryParameters } from 'sveltekit-search-params';

	import SelectInputAny from '@/components/ui-custom/select-input-any.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator/index.js';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../../+layout@.svelte';
	import {
		ALL_WORKFLOW_STATUSES,
		isExtendedWorkflowStatus,
		parseLimit,
		parseOffset
	} from '../../tests/runs/_partials/index.js';
	import PaginationArrows from '../../tests/runs/_partials/pagination-arrows.svelte';

	//

	let { data } = $props();
	let { pipeline, pagination } = $derived(data);

	$effect(() => {
		setDashboardNavbar({ title: m.Pipelines() });
	});

	const params = queryParameters({
		status: {
			encode: (value) => value,
			decode: (value) => {
				if (isExtendedWorkflowStatus(value)) return value;
				return undefined;
			}
		},
		limit: {
			encode: (value) => value,
			decode: parseLimit
		},
		offset: {
			encode: (value) => value,
			decode: parseOffset
		}
	});

	const statusOptions = $derived([
		{ value: undefined, label: m.All() },
		...ALL_WORKFLOW_STATUSES.filter((status) => status !== null).map((status) => ({
			value: status,
			label: status
		}))
	]);

	const workflows = new PolledResource(
		() =>
			Pipeline.Workflows.list(pipeline.id, {
				status: params.status,
				limit: params.limit ?? undefined,
				offset: params.offset ?? undefined
			}),
		{
			initialValue: () => data.workflows,
			intervalMs: 10000
		}
	);

	//

	// Filter out canceled runs for success rate calculation
	const nonCanceledWorkflows = $derived(
		(workflows.current ?? []).filter((w) => w.status !== 'Canceled')
	);

	const totalRuns = $derived(nonCanceledWorkflows.length);

	const totalSuccesses = $derived(
		nonCanceledWorkflows.filter((w) => w.status === 'Completed').length
	);

	const successRate = $derived(((totalSuccesses / totalRuns) * 100).toFixed(1) + '%');
	const currentItemCount = $derived(workflows.current?.length ?? 0);
</script>

<div class="flex items-end justify-between gap-8">
	<div class="space-y-2">
		<BackButton href="/my/pipelines" class="px-0!" />

		<div>
			<T class="text-muted-foreground">{m.Pipeline()}</T>
			<T tag="h2">{pipeline.name}</T>
		</div>
	</div>

	<div class="flex flex-wrap gap-2 md:flex-nowrap">
		{@render numberBox(totalRuns, m.Total_runs())}
		{@render numberBox(totalSuccesses, m.Successful_runs())}
		{@render numberBox(successRate, m.Success_rate())}
	</div>
</div>

<Separator />

<div class="flex flex-wrap items-center justify-between gap-3">
	<T tag="h3">{m.workflow_runs()}</T>

	<div class="flex flex-wrap items-center gap-3">
		<SelectInputAny
			items={statusOptions}
			value={params.status}
			placeholder={m.Status()}
			onValueChange={(value) => {
				params.status = value;
				params.offset = 0;
			}}
		/>

		<PaginationArrows
			pagination={{
				limit: params.limit ?? pagination.limit,
				offset: params.offset ?? pagination.offset
			}}
			{currentItemCount}
			onPrevious={() => {
				params.offset = Math.max((params.offset ?? 0) - 1, 0);
			}}
			onNext={() => {
				params.offset = (params.offset ?? 0) + 1;
			}}
			onLimitChange={(limit) => {
				params.limit = limit;
				params.offset = 0;
			}}
		/>
	</div>
</div>
<Pipeline.Workflows.Table workflows={workflows.current ?? []} hidePipelineColumn />

<!--  -->

{#snippet numberBox(highlight: string | number, description: string)}
	<div class="flex h-20 w-[140px] flex-col items-start justify-between rounded-lg border p-3">
		<T tag="h2" class="mb-0! pb-0!">{highlight}</T>
		<T class="text-sm">{description}</T>
	</div>
{/snippet}
