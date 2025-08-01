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
	import { Separator } from '@/components/ui/separator';
	import Button from '@/components/ui-custom/button.svelte';
	import { ArrowLeftIcon } from 'lucide-svelte';

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

{#if !isIframeLoading}
	<div class="bg-primary">
		<div class="padding-x">
			<BackButton href="/my/tests/runs" class="text-white">
				{m.Back_to_test_runs()}
			</BackButton>
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
						{#each testNameChunks as chunk, index}
							{#if index > 0}
								<span class="text-muted-foreground">:</span>
							{/if}
							<span>
								{chunk}
							</span>
						{/each}
					</T>
				{:else}
					<T tag="h3" class="break-words !p-0">
						{execution.id}
					</T>
				{/if}
			</div>

			{#if execution.status}
				<TemporalI18nProvider>
					<WorkflowStatus status={execution.status} />
				</TemporalI18nProvider>
			{/if}

			<table class="mt-6 text-sm">
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
				</tbody>
			</table>
		</div>

		{#if execution.status === 'Running'}
			<WorkflowQrPoller {workflowId} {runId} containerClass="size-40" />
		{/if}
	</div>

	<div class="bg-temporal padding-x py-2">
		<Separator />
	</div>

	{#if memo}
		{#if memo.author == 'ewc'}
			<EwcTop {workflowId} namespace={organization?.id!} />
		{:else}
			<div class="bg-temporal padding-x space-y-8 pt-4">
				{#if memo.author == 'openid_conformance_suite'}
					<OpenidnetTop {workflowId} {runId} namespace={organization?.id!} />
				{:else if memo.author == 'eudiw'}
					<EudiwTop {workflowId} namespace={organization?.id!} />
				{/if}

				<Separator />
			</div>
		{/if}
	{/if}
{/if}

<iframe
	id={iframeId}
	title="Workflow"
	src={page.url.pathname + '/temporal'}
	class="w-full overflow-hidden"
></iframe>

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
				<Button
					variant="link"
					class="!px-0 text-black"
					onclick={() => window.history.back()}
				>
					<ArrowLeftIcon />
					{m.Back()}
				</Button>
			</div>
		</div>
	{/snippet}
</LoadingDialog>

<style lang="postcss">
	.bg-temporal {
		background-color: rgb(248 250 252);
	}

	.padding-x {
		@apply !px-2 md:!px-4 lg:!px-8;
	}
</style>
