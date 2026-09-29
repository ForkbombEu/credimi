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

	type Props = {
		open?: boolean;
		workflowId: string;
		runId: string;
		deviceId: string | undefined;
		deviceName: string | undefined;
	};

	type LiveViewOutcome = { ok: true; url: string } | { ok: false; error: string };

	let { open = $bindable(false), workflowId, runId, deviceId, deviceName }: Props = $props();

	/** Fresh promise whenever the sheet opens or the device/run identity changes. */
	const liveViewRequest = $derived.by((): Promise<LiveViewOutcome> | undefined => {
		if (!open || !deviceId) return undefined;

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
	side="right"
	title={m.Live_view()}
	class="h-full !w-[min(100vw,24rem)]"
	contentClass="flex h-full min-h-0 flex-1 flex-col !overflow-hidden !px-0"
>
	{#snippet content()}
		{#if deviceName}
			<p class="shrink-0 truncate px-6 pb-2 font-mono text-sm text-muted-foreground">
				{deviceName}
			</p>
		{/if}

		{#if liveViewRequest}
			{#await liveViewRequest}
				<div
					class="flex min-h-[50vh] flex-1 flex-col items-center justify-center gap-3 px-6 py-8"
				>
					<Spinner />
					<p class="text-sm text-muted-foreground">{m.Please_wait()}</p>
				</div>
			{:then outcome}
				{#if outcome.ok}
					<div class="flex shrink-0 items-center justify-end px-6 pb-2">
						<Button
							variant="outline"
							size="sm"
							href={outcome.url}
							target="_blank"
							rel="noopener noreferrer"
							class="gap-1.5"
						>
							<ExternalLinkIcon class="size-4" />
							{m.Open_in_new_page()}
						</Button>
					</div>
					<div class="min-h-0 flex-1 overflow-hidden">
						{#key deviceId}
							<iframe
								src={outcome.url}
								title={iframeTitle}
								class="h-full w-full border-0"
								allow="autoplay"
							></iframe>
						{/key}
					</div>
				{:else}
					<div class="flex min-h-[50vh] flex-1 flex-col gap-2 px-6 py-4">
						<p class="text-sm">{m.Live_view_open_failed()}</p>
						<p class="font-mono text-xs break-words text-muted-foreground">
							{outcome.error}
						</p>
					</div>
				{/if}
			{:catch}
				<div class="flex min-h-[50vh] flex-1 flex-col gap-2 px-6 py-4">
					<p class="text-sm">{m.Live_view_open_failed()}</p>
				</div>
			{/await}
		{/if}
	{/snippet}
</Sheet>
