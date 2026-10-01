<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowStatus as WorkflowStatusValue } from '$lib/workflows/types';

	import { WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { Workflow } from '$lib';
	import BackButton from '$lib/layout/back-button.svelte';
	import { runWithLoading } from '$lib/layout/global-loading.svelte';
	import ExecutionDevices from '$lib/pipeline/execution-devices.svelte';
	import { getExecutionDevices, type ExecutionSummary } from '$lib/pipeline/workflows';
	import { formatExecutionTimestamp } from '$lib/scoreboard/extras/format-date';
	import { TemporalI18nProvider } from '$lib/temporal';
	import { isOpenIDConformanceStandard } from '$lib/wallet-test-pages/openidnet';
	import { WorkflowQrPoller } from '$lib/workflows';
	import { onDestroy, untrack } from 'svelte';
	import { fromStore } from 'svelte/store';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import FailureText from '@/components/ui-custom/failure-text.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import EudiwTop from './_partials/eudiw-top.svelte';
	import EwcTop from './_partials/ewc-top.svelte';
	import OpenidnetTop from './_partials/openidnet-top.svelte';
	import { _getWorkflow } from './+layout';

	//

	let { data } = $props();
	let { organization, workflow } = $derived(data);
	let { memo, execution, devices } = $derived(workflow);
	let { id: workflowId, runId } = $derived(execution);

	const user = fromStore(currentUser);
	const timezone = $derived(user.current?.Timezone);
	const startDisplay = $derived(formatExecutionTimestamp(execution.startTime, timezone) ?? '-');
	const endDisplay = $derived(formatExecutionTimestamp(execution.endTime, timezone) ?? '-');

	/** Adapt Temporal describe payload into the pipeline ExecutionSummary shape. */
	const executionSummary = $derived<ExecutionSummary>({
		execution: {
			workflowId: execution.id,
			runId: execution.runId
		},
		type: { name: execution.name },
		startTime: execution.startTime ? String(execution.startTime) : '',
		...(execution.endTime != null ? { endTime: String(execution.endTime) } : {}),
		status: execution.status as WorkflowStatusValue,
		displayName: execution.id,
		devices
	});
	const executionDevices = $derived(getExecutionDevices(executionSummary));

	/* Embedded Temporal UI */

	// Upstream Temporal UI behind Credimi's read-only proxy scoped to the user's organization
	// namespace (pkg/internal/temporalui). The proxy authenticates the `pb_auth` cookie that
	// hooks.client.ts keeps in sync with the auth store.
	const temporalUiUrl = $derived(
		[
			'/temporal-ui/namespaces',
			encodeURIComponent(organization.canonified_name),
			'workflows',
			encodeURIComponent(workflowId),
			encodeURIComponent(runId),
			'timeline'
		].join('/')
	);
	let loadedTemporalUiUrl = $state<string>();
	const isTemporalUiLoading = $derived(loadedTemporalUiUrl !== temporalUiUrl);
	let temporalUiIframe = $state<HTMLIFrameElement>();
	// Set on iframe onload for the current src; height messages from a previous
	// document must not clear the spinner for a newly navigated URL.
	let temporalUiOnloadUrl = $state<string>();
	// Same guard as the pre-embed iframe (#497): ignore the steady delta that
	// appears when the iframe height feeds back into the child's scrollHeight.
	let temporalUiHeightDelta = $state(0);
	// onload alone can reveal a tall empty iframe before the embed posts height;
	// fall back if that message never arrives (e.g. blocked script).
	const TEMPORAL_UI_LOAD_FALLBACK_MS = 1500;
	let temporalUiLoadFallback: ReturnType<typeof setTimeout> | undefined;

	function markTemporalUiReady(url: string) {
		if (loadedTemporalUiUrl === url) return;
		loadedTemporalUiUrl = url;
		if (temporalUiLoadFallback !== undefined) {
			clearTimeout(temporalUiLoadFallback);
			temporalUiLoadFallback = undefined;
		}
	}

	function onTemporalUiMessage(ev: MessageEvent) {
		if (ev.origin !== window.location.origin) return;
		const data = ev.data as { source?: string; type?: string; height?: number } | null;
		if (!data || data.source !== 'credimi-temporal-ui' || data.type !== 'height') return;
		const iframe = temporalUiIframe;
		if (!iframe || typeof data.height !== 'number' || data.height <= 0) return;
		const next = Math.ceil(data.height);
		const delta = next - (parseInt(iframe.height, 10) || 0);
		if (delta !== temporalUiHeightDelta) {
			iframe.height = `${next}px`;
			temporalUiHeightDelta = delta;
		}
		// First paint-sized height after onload: content is measurable.
		if (temporalUiOnloadUrl === temporalUiUrl) {
			markTemporalUiReady(temporalUiUrl);
		}
	}

	onDestroy(() => {
		if (temporalUiLoadFallback !== undefined) {
			clearTimeout(temporalUiLoadFallback);
			temporalUiLoadFallback = undefined;
		}
	});

	/* Run status refresh */

	const POLL_INTERVAL_MS = 5000;

	$effect(() => {
		const run = { workflowId, runId };
		return untrack(() => pollRunWhileRunning(run.workflowId, run.runId));
	});

	// Sequential: a slow describe must not overlap the next one.
	function pollRunWhileRunning(polledWorkflowId: string, polledRunId: string) {
		let active = true;
		let timer: ReturnType<typeof setTimeout> | undefined;

		function schedule() {
			if (active && workflow.execution.status === 'Running') {
				timer = setTimeout(poll, POLL_INTERVAL_MS);
			}
		}

		async function poll() {
			const w = await _getWorkflow(polledWorkflowId, polledRunId);
			if (!active) return;
			if (w instanceof Error) console.error(w);
			else workflow = w;
			schedule();
		}

		schedule();

		return () => {
			active = false;
			clearTimeout(timer);
		};
	}

	/* UI */

	const testNameChunks = $derived(memo?.test.split('+') ?? []);
	const openIDConformanceStandard = $derived(
		memo?.author === 'openid_conformance_suite' && isOpenIDConformanceStandard(memo.standard)
			? memo.standard
			: undefined
	);

	const failureMessage = $derived(
		(execution as typeof execution & { failure_reason?: string }).failure_reason
	);
</script>

<svelte:head>
	<style>
		body {
			background-color: rgb(248 250 252);
		}
	</style>
</svelte:head>

<svelte:window onmessage={onTemporalUiMessage} />

<div class="bg-primary">
	<div class="padding-x">
		<BackButton href="/my/tests/runs" class="text-white" />
	</div>
</div>

<div
	class="bg-temporal padding-x flex flex-wrap items-start justify-between gap-4 py-4 pb-4 sm:flex-nowrap sm:gap-8"
>
	<div>
		<div class="mb-4">
			{#if memo}
				<T tag="h3">
					{memo?.standard} / {memo?.author}
				</T>
				<T tag="h1">
					{#each testNameChunks as chunk, index (index)}
						{#if index > 0}
							<span class="text-muted-foreground">:</span>
						{/if}
						<span>
							{chunk}
						</span>
					{/each}
				</T>
			{:else}
				<T tag="h3" class="!p-0 break-words">
					{execution.id}
				</T>
			{/if}
		</div>

		{#if execution.status}
			<TemporalI18nProvider>
				<WorkflowStatus status={execution.status} />
			</TemporalI18nProvider>
		{/if}

		{#if failureMessage}
			<Alert variant="destructive" class="mt-2 block p-3! text-sm">
				<span class="font-bold">{m.reason()}:</span>
				<FailureText>{failureMessage}</FailureText>
			</Alert>
		{/if}

		<table class="mt-6 text-sm">
			<tbody>
				<tr>
					<td class="italic"> Start </td>
					<td class="pl-4 font-mono">
						{startDisplay}
					</td>
				</tr>
				<tr>
					<td class="italic"> End </td>
					<td class="pl-4 font-mono">
						{endDisplay}
					</td>
				</tr>
				<tr>
					<td class="h-2"></td>
				</tr>
				<tr>
					<td class="italic"> Workflow ID </td>
					<td class="pl-4">
						{execution.id}
					</td>
				</tr>
				<tr>
					<td class="italic"> Run ID </td>
					<td class="pl-4">
						{execution.runId}
					</td>
				</tr>
				{#if executionDevices.length > 0}
					<tr>
						<td class="h-2"></td>
					</tr>
					<tr>
						<td class="align-top italic"> Devices </td>
						<td class="pl-4">
							<ExecutionDevices execution={executionSummary} />
						</td>
					</tr>
				{/if}
			</tbody>
		</table>

		<div class="flex flex-wrap gap-2 pt-6">
			<Button
				variant="outline"
				onclick={() => runWithLoading({ fn: () => Workflow.cancel(workflowId, runId) })}
				disabled={execution.status !== 'Running'}
			>
				{m.Cancel()}
			</Button>
		</div>
	</div>

	{#if workflow.execution.name !== 'Dynamic Pipeline Workflow'}
		<WorkflowQrPoller {workflowId} {runId} showQrLink={true} containerClass="size-40" />
	{/if}
</div>

<div class="bg-temporal padding-x py-2">
	<Separator />
</div>

{#if memo}
	{#if memo.author == 'ewc' || memo.author == 'webuild'}
		<EwcTop {workflowId} namespace={organization.canonified_name} />
	{:else}
		<div class="bg-temporal padding-x space-y-8 pt-4">
			{#if memo.author == 'openid_conformance_suite'}
				{#if openIDConformanceStandard}
					<OpenidnetTop
						{workflowId}
						namespace={organization.canonified_name}
						standard={openIDConformanceStandard}
					/>
				{/if}
			{:else if memo.author == 'eudiw'}
				<EudiwTop {workflowId} namespace={organization.canonified_name} />
			{/if}

			<Separator />
		</div>
	{/if}
{/if}

<div class="relative" aria-busy={isTemporalUiLoading}>
	{#if isTemporalUiLoading}
		<div class="bg-temporal padding-x absolute inset-0 z-10 pt-4">
			<div
				class={[
					'rounded-lg border bg-slate-200 py-10 text-center',
					'flex items-center justify-center gap-2',
					'animate-pulse'
				]}
			>
				<Spinner size={16} />
				<T class="text-muted-foreground">
					{m.Loading_workflow_data_may_take_some_seconds()}
				</T>
			</div>
		</div>
	{/if}

	<iframe
		bind:this={temporalUiIframe}
		title="Temporal workflow history"
		src={temporalUiUrl}
		class="block min-h-[600px] w-full max-w-full border-0"
		style="overflow: hidden;"
		scrolling="no"
		onload={() => {
			const url = temporalUiUrl;
			temporalUiOnloadUrl = url;
			temporalUiHeightDelta = 0;
			if (temporalUiLoadFallback !== undefined) clearTimeout(temporalUiLoadFallback);
			// Prefer first height message; clear spinner anyway if none arrives.
			temporalUiLoadFallback = setTimeout(() => {
				markTemporalUiReady(url);
			}, TEMPORAL_UI_LOAD_FALLBACK_MS);
		}}
	></iframe>
</div>

<style lang="postcss">
	@reference 'tailwindcss';

	.bg-temporal {
		background-color: rgb(248 250 252);
	}

	.padding-x {
		@apply px-2! md:px-4! lg:px-8!;
	}
</style>
