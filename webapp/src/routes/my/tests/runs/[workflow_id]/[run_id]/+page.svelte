<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import BackButton from '$lib/layout/back-button.svelte';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { onMount } from 'svelte';
	import OpenidnetTop from './_partials/openidnet-top.svelte';
	import EwcTop from './_partials/ewc-top.svelte';
	import EudiwTop from './_partials/eudiw-top.svelte';
	import { WorkflowQrPoller } from '$lib/workflows';
	import {
		setupEmitter,
		setupListener,
		type PageMessage,
		type IframeMessage
	} from './_partials/page-events';
	import { toUserTimezone } from '@/utils/toUserTimezone';
	import { _getWorkflow } from './+layout';
	import { WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { TemporalI18nProvider } from '$lib/temporal';

	//

	let { data } = $props();
	let { organization, workflow } = $derived(data);
	let { execution, memo } = $derived(workflow);
	let { id: workflowId, runId } = $derived(execution);

	/* Iframe communication */

	const iframeId = 'iframe';

	function getIframe() {
		if (!browser) return;
		const iframe = document.getElementById(iframeId);
		if (!iframe || !(iframe instanceof HTMLIFrameElement)) return;
		return iframe;
	}

	function getIframeWindow() {
		const iframe = getIframe();
		return iframe?.contentWindow ?? undefined;
	}

	//

	let isIframeLoading = $state(true);

	setupListener<IframeMessage>((ev) => {
		if (ev.type === 'height') {
			const iframe = getIframe();
			if (iframe) iframe.height = ev.height + 'px';
		}
		if (ev.type === 'ready') {
			isIframeLoading = false;
		}
	});

	const emit = setupEmitter<PageMessage>(getIframeWindow);

	// Sending workflow data to iframe

	onMount(() => {
		emit({
			type: 'workflow',
			...workflow
		});

		const interval = setInterval(async () => {
			const w = await _getWorkflow(workflowId, runId);
			if (w instanceof Error) {
				console.error(w);
			} else {
				emit({
					type: 'workflow',
					...w
				});
			}
		}, 5000);

		return () => {
			clearInterval(interval);
		};
	});

	/* UI */

	const testNameChunks = $derived(memo?.test.split(':') ?? []);
</script>

<div class="min-h-screen">
	{#if !isIframeLoading}
		<div class="bg-primary">
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
				{#if memo}
					<T tag="h3">
						{memo?.standard} / {memo?.author}
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
					<T class="mt-4">{m.Test_run()}: {execution.id}</T>
				{:else}
					<div>
						<T tag="h2">
							{execution.id}
						</T>
					</div>
				{/if}

				<div class="space-y-4 border-b-2 px-2 py-4 md:px-4 lg:px-8">
					{#if execution.status}
						<TemporalI18nProvider>
							<WorkflowStatus status={execution.status} />
						</TemporalI18nProvider>
					{/if}

					<table>
						<tbody>
							<tr>
								<td class="italic"> Start </td>
								<td class="pl-4">
									{toUserTimezone(execution.startTime) ?? '-'}
								</td>
							</tr>
							<tr>
								<td class="italic"> End </td>
								<td class="pl-4">
									{toUserTimezone(execution.endTime) ?? '-'}
								</td>
							</tr>
							<tr>
								<td class="italic"> Elapsed </td>
								<td class="pl-4">
									{execution.executionTime ?? '-'}
								</td>
							</tr>
							<tr>
								<td class="italic"> Run ID </td>
								<td class="pl-4">
									{execution.runId}
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
			<div class="bg-secondary rounded-md p-4">
				<WorkflowQrPoller {workflowId} {runId} containerClass="size-40" />
			</div>
		</div>

		{#if memo}
			<div class="border-b-2 border-b-black !px-2 py-4 md:!px-4 lg:!px-8">
				{#if memo.author == 'openid_conformance_suite'}
					<OpenidnetTop {workflowId} {runId} namespace={organization?.id!} />
				{:else if memo.author == 'ewc'}
					<EwcTop {workflowId} {runId} namespace={organization?.id!} />
				{:else if memo.author == 'eudiw'}
					<EudiwTop {workflowId} namespace={organization?.id!} />
				{/if}
			</div>
		{/if}
	{/if}

	<iframe
		id={iframeId}
		title="Workflow"
		src={page.url.pathname + '/temporal'}
		class="w-full overflow-hidden"
	></iframe>
</div>

<LoadingDialog
	loading={isIframeLoading}
	contentClass="p-0 pt-6 gap-4 overflow-hidden !max-w-[300px]"
>
	<div class="px-4">
		<T class="text-center">{m.Loading_workflow_data_may_take_some_seconds()}</T>
	</div>

	{#snippet bottom()}
		<div class="w-full pt-2">
			<div class="w-full bg-gray-200 px-4">
				<BackButton href="/my/tests/runs" class="text-black">
					{m.Back_to_test_runs()}
				</BackButton>
			</div>
		</div>
	{/snippet}
</LoadingDialog>
