<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import BackButton from '$lib/layout/back-button.svelte';
	import { TemporalI18nProvider } from '$lib/temporal';
	import { WorkflowQrPoller } from '$lib/workflows';
	import { onMount } from 'svelte';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { toUserTimezone } from '@/utils/toUserTimezone';

	import EudiwTop from './_partials/eudiw-top.svelte';
	import EwcTop from './_partials/ewc-top.svelte';
	import OpenidnetTop from './_partials/openidnet-top.svelte';
	import {
		setupEmitter,
		setupListener,
		type IframeMessage,
		type PageMessage
	} from './_partials/page-events';
	import { _getWorkflow } from './+layout';

	//

	let { data } = $props();
	let { organization, workflow } = $derived(data);
	let { memo, execution } = $derived(workflow);
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
	let constantHeightDifference = $state(0);

	setupListener<IframeMessage>((ev) => {
		if (ev.type === 'height') {
			const iframe = getIframe();
			if (!iframe) return;
			const heightDifference = ev.height - (parseInt(iframe.height) || 0);
			if (heightDifference !== constantHeightDifference) {
				iframe.height = ev.height + 'px';
				constantHeightDifference = heightDifference;
			}
		} else if (ev.type === 'ready') {
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
				workflow = w;
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

	const testNameChunks = $derived(memo?.test.split('+') ?? []);
</script>

<svelte:head>
	<style>
		body {
			background-color: rgb(248 250 252);
		}
	</style>
</svelte:head>

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

	<WorkflowQrPoller {workflowId} {runId} showQrLink={true} containerClass="size-40" />
</div>

<div class="bg-temporal padding-x py-2">
	<Separator />
</div>

{#if memo}
	{#if memo.author == 'ewc'}
		<EwcTop {workflowId} namespace={organization.canonified_name} />
	{:else}
		<div class="bg-temporal padding-x space-y-8 pt-4">
			{#if memo.author == 'openid_conformance_suite'}
				<OpenidnetTop {workflowId} {runId} namespace={organization.canonified_name} />
			{:else if memo.author == 'eudiw'}
				<EudiwTop {workflowId} namespace={organization.canonified_name} />
			{/if}

			<Separator />
		</div>
	{/if}
{/if}

<div class="relative min-h-[500px]">
	{#if isIframeLoading}
		<div class="bg-temporal padding-x absolute inset-0 pt-4">
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
		id={iframeId}
		title="Workflow"
		src={page.url.pathname + '/temporal'}
		class="bg-animate-pulse w-full"
		style="overflow: hidden;"
		scrolling="no"
	></iframe>
</div>

<style lang="postcss">
	.bg-temporal {
		background-color: rgb(248 250 252);
	}

	.padding-x {
		@apply !px-2 md:!px-4 lg:!px-8;
	}
</style>
