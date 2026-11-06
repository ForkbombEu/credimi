<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import { ExternalLinkIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import { m } from '@/i18n';

	import { requestLiveView } from './live-view';

	type Props = {
		workflowId: string;
		runId: string;
		deviceId: string;
		deviceName?: string;
		/** Hides the runner page's Maestro commands panel, leaving only the device screen. */
		hideCommands?: boolean;
		class?: ClassValue;
	};

	type LiveViewOutcome = { ok: true; url: string } | { ok: false; error: string };

	let {
		workflowId,
		runId,
		deviceId,
		deviceName,
		hideCommands = false,
		class: className
	}: Props = $props();

	// A derived string that recomputes to the same value doesn't notify: callers that poll and
	// pass fresh but equal props don't open a new stream.
	const target = $derived(`${workflowId}\n${runId}\n${deviceId}`);

	const request = $derived.by((): Promise<LiveViewOutcome> => {
		const [w, r, d] = target.split('\n');
		return requestLiveView(w, r, d).then((result): LiveViewOutcome => {
			if (result.isErr) {
				return { ok: false, error: result.error };
			}

			const stream = result.value.find((s) => s.device_id === d) ?? result.value[0];
			if (!stream?.url) {
				return { ok: false, error: m.Live_view_open_failed() };
			}

			return { ok: true, url: stream.url };
		});
	});

	function frameUrl(url: string): string {
		if (!hideCommands) return url;
		const withQuery = new URL(url);
		withQuery.searchParams.set('commands', 'off');
		return withQuery.toString();
	}

	const iframeTitle = $derived(deviceName ? `${m.Live_view()}: ${deviceName}` : m.Live_view());
</script>

{#await request}
	<div class="grid min-h-[50dvh] place-content-center justify-items-center gap-3">
		<Spinner />
		<p class="text-sm text-muted-foreground">{m.Please_wait()}</p>
	</div>
{:then outcome}
	{#if outcome.ok}
		{@const src = frameUrl(outcome.url)}
		<div class="flex flex-col items-end">
			<Button
				variant="link"
				size="sm"
				href={src}
				target="_blank"
				rel="noopener noreferrer"
				class="px-0"
			>
				<ExternalLinkIcon />
				{m.Open_in_new_page()}
			</Button>

			<!-- The runner page lays out the device screen and, unless hidden, its Maestro commands side by side. -->
			<iframe
				{src}
				title={iframeTitle}
				class={['w-full border-0', className]}
				allow="autoplay"
			></iframe>
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
