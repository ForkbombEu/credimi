<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ExternalLinkIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import { m } from '@/i18n';

	import { requestLiveView } from './live-view';
	import { getExecutionDevices, type ExecutionSummary } from './workflows';

	type Props = {
		open?: boolean;
		execution: ExecutionSummary;
		deviceId: string | undefined;
	};

	type LiveViewOutcome = { ok: true; url: string } | { ok: false; error: string };

	let { open = $bindable(false), execution, deviceId }: Props = $props();

	const workflowId = $derived(execution.execution.workflowId);
	const runId = $derived(execution.execution.runId);
	const deviceName = $derived(
		deviceId
			? getExecutionDevices(execution).find((d) => d.device_id === deviceId)?.name
			: undefined
	);

	/** Fresh promise whenever the sheet opens or the device/run identity changes. */
	const liveViewRequest = $derived.by((): Promise<LiveViewOutcome> | undefined => {
		if (!open || !deviceId || !workflowId || !runId) return undefined;

		const id = deviceId;
		return requestLiveView(workflowId, runId, id).then((result): LiveViewOutcome => {
			if (result.isErr) {
				return { ok: false, error: result.error };
			}

			const stream = result.value.find((s) => s.device_id === id) ?? result.value[0];
			if (!stream?.url) {
				return { ok: false, error: m.Live_view_open_failed() };
			}

			return { ok: true, url: stream.url };
		});
	});

	const iframeTitle = $derived(deviceName ? `${m.Live_view()}: ${deviceName}` : m.Live_view());
</script>

<Sheet
	bind:open
	hideTrigger
	title={m.Live_view()}
	class="!w-[min(100vw,24rem)]"
	contentClass="flex min-h-0 flex-1 flex-col gap-4"
>
	{#snippet content()}
		{#if liveViewRequest}
			<div class="space-y-3 text-sm">
				<div>
					<p class="font-semibold">{execution.displayName}</p>
					{#if deviceName}
						<p>{m.Device()}: {deviceName}</p>
					{/if}
				</div>
			</div>

			{#await liveViewRequest}
				<div class="grid min-h-[50dvh] place-content-center justify-items-center gap-3">
					<Spinner />
					<p class="text-sm text-muted-foreground">{m.Please_wait()}</p>
				</div>
			{:then outcome}
				{#if outcome.ok}
					<div class="flex flex-col items-end">
						<Button
							variant="link"
							size="sm"
							href={outcome.url}
							target="_blank"
							rel="noopener noreferrer"
							class="px-0"
						>
							<ExternalLinkIcon />
							{m.Open_in_new_page()}
						</Button>

						{#key deviceId}
							<iframe
								src={outcome.url}
								title={iframeTitle}
								class="aspect-[9/19.5] w-full border-0"
								allow="autoplay"
							></iframe>
						{/key}
					</div>
				{:else}
					<div class="space-y-2">
						<p class="text-sm">{m.Live_view_open_failed()}</p>
						<p class="font-mono text-xs break-words text-muted-foreground">
							{outcome.error}
						</p>
					</div>
				{/if}
			{:catch}
				<p class="text-sm">{m.Live_view_open_failed()}</p>
			{/await}

			<dl class="space-y-3 text-sm text-muted-foreground">
				<div>
					<dt>{m.Workflow_ID()}</dt>
					<dd class="truncate font-mono text-xs">{workflowId}</dd>
				</div>
				<div>
					<dt>Run ID</dt>
					<dd class="truncate font-mono text-xs">{runId}</dd>
				</div>
			</dl>
		{/if}
	{/snippet}
</Sheet>
