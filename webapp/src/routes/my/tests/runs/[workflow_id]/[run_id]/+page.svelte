<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { z } from 'zod';
	import type { HistoryEvent } from '@forkbombeu/temporal-ui';
	import { _loadData, type WorkflowResponse } from './+layout';

	export const WorkflowMessageSchema = z.object({
		type: z.literal('workflow'),
		workflow: z.custom<WorkflowResponse>(),
		eventHistory: z.custom<HistoryEvent[]>()
	});
</script>

<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import BackButton from '$lib/layout/back-button.svelte';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { onDestroy, onMount } from 'svelte';
	import { HeightMessageSchema } from './temporal/+page.svelte';
	import OpenidnetTop from './_partials/openidnet-top.svelte';
	import EwcTop from './_partials/ewc-top.svelte';
	import EudiwTop from './_partials/eudiw-top.svelte';
	import { WorkflowQrPoller } from '$lib/workflows';

	//

	let { data } = $props();
	let { workflowId, runId, workflowMemo, organization, workflow, eventHistory } = $derived(data);

	const testNameChunks = $derived(workflowMemo?.test.split(':') ?? []);

	/* Iframe communication */

	const iframeId = 'iframe';

	// Sending workflow data to iframe

	onMount(() => {
		const interval = setInterval(async () => {
			const data = await _loadData(workflowId, runId, { fetch });
			if (data instanceof Error) {
				console.error(data);
			} else {
				workflow = data.workflow;
				eventHistory = data.eventHistory;
			}
		}, 5000);

		return () => {
			clearInterval(interval);
		};
	});

	$effect(() => {
		const iframe = document.getElementById(iframeId);
		if (!iframe || !(iframe instanceof HTMLIFrameElement)) return;

		iframe.contentWindow?.postMessage(
			{
				type: 'workflow',
				workflow: $state.snapshot(workflow),
				eventHistory: $state.snapshot(eventHistory)
			},
			'*'
		);
	});

	// Loading handling and height calculation

	let isIframeLoading = $state(true);

	function onMessage(event: MessageEvent) {
		const message = HeightMessageSchema.safeParse(event.data);
		if (!message.success) return;

		const iframe = document.getElementById(iframeId);
		if (!iframe || !(iframe instanceof HTMLIFrameElement)) return;

		iframe.height = message.data.height + 'px';
		isIframeLoading = false;
	}

	if (browser) {
		window.addEventListener('message', onMessage);

		onDestroy(() => {
			window.removeEventListener('message', onMessage);
		});
	}
</script>

<div class="min-h-screen">
	<div class="bg-primary text-white">
		<div class="!px-2 md:!px-4 lg:!px-8">
			<BackButton href="/my/tests/runs" class="text-white">
				{m.Back_to_test_runs()}
			</BackButton>
		</div>
	</div>

	<div
		class="border-primary flex items-center justify-between border-b-2 !px-2 py-4 pb-4 md:!px-4 lg:!px-8"
	>
		<div>
			{#if workflowMemo}
				<T tag="h3">
					{workflowMemo?.standard} / {workflowMemo?.author}
				</T>
				<T tag="h1">
					{#each testNameChunks as chunk, index}
						{#if index > 0}
							<span class="text-muted-foreground">:</span>
						{/if}
						<span>
							{chunk}
						</span>
					{/each}
				</T>
				<T class="mt-4">{m.Test_run()}: {workflowId}</T>
			{:else}
				<div>
					<T tag="h2">
						{workflowId}
					</T>
				</div>
			{/if}
		</div>
		<div class="bg-secondary rounded-md p-4">
			<WorkflowQrPoller {workflowId} {runId} containerClass="size-40" />
		</div>
	</div>

	{#if workflowMemo}
		<div class="border-b-2 border-b-black !px-2 py-4 md:!px-4 lg:!px-8">
			{#if workflowMemo.author == 'openid_conformance_suite'}
				<OpenidnetTop {workflowId} {runId} namespace={organization?.id!} />
			{:else if workflowMemo.author == 'ewc'}
				<EwcTop {workflowId} {runId} namespace={organization?.id!} />
			{:else if workflowMemo.author == 'eudiw'}
				<EudiwTop {workflowId} namespace={organization?.id!} />
			{/if}
		</div>
	{/if}

	<iframe id={iframeId} title="Workflow" src={page.url.pathname + '/temporal'} class="w-full"
	></iframe>
</div>

<LoadingDialog loading={isIframeLoading} />
