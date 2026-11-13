<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CheckIcon, SaveIcon, WrenchIcon, XIcon } from '@lucide/svelte';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';

	import Alert from '@/components/ui-custom/alert.svelte';
	import RenderMD from '@/components/ui-custom/renderMD.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import { getTranscript, type TranscriptEntry } from './api';
	import { formatTranscriptTime } from './transcript-time';

	type Props = {
		sessionId: string;
		timezone?: string;
	};

	const { sessionId, timezone }: Props = $props();

	let entries = $state<TranscriptEntry[]>();
	let loadError = $state('');

	// A closed chat does not change: it is read once per session.
	$effect(() => {
		const id = sessionId;
		let active = true;
		entries = undefined;
		loadError = '';
		void getTranscript(id).then((res) => {
			if (!active) return;
			if (res.isOk) entries = res.value;
			else loadError = res.error.message;
		});
		return () => {
			active = false;
		};
	});
</script>

{#snippet time(entry: TranscriptEntry)}
	<time datetime={entry.time} class="shrink-0 font-mono text-xs text-muted-foreground">
		{formatTranscriptTime(entry.time, timezone)}
	</time>
{/snippet}

<div class="min-h-0 flex-1 overflow-y-auto p-4" role="log" aria-label={m.Chat_transcript()}>
	{#if loadError}
		<Alert variant="destructive" class="block border border-destructive p-3!">
			{loadError}
		</Alert>
	{:else if !entries}
		<div class="flex items-center justify-center gap-2 p-6">
			<Spinner size={16} />
			<T class="text-muted-foreground">{m.Loading()}</T>
		</div>
	{:else if entries.length === 0}
		<p class="text-sm text-muted-foreground">{m.Transcript_empty()}</p>
	{:else}
		<ol class="mx-auto flex max-w-3xl flex-col gap-4">
			{#each entries as entry, i (i)}
				<li>
					{#if entry.kind === 'user'}
						<div class="flex flex-col items-end gap-1">
							<p
								class="max-w-[85%] rounded-2xl bg-secondary px-3 py-2 text-sm wrap-anywhere whitespace-pre-wrap"
							>
								{entry.text}
							</p>
							{@render time(entry)}
						</div>
					{:else if entry.kind === 'assistant'}
						<div class="flex flex-col items-start gap-1">
							<RenderMD
								content={entry.text ?? ''}
								class="prose prose-sm max-w-none prose-a:text-primary"
							/>
							{@render time(entry)}
						</div>
					{:else if entry.kind === 'tool_call'}
						<div
							class="rounded-md bg-slate-50 px-3 py-1.5 text-xs text-muted-foreground"
						>
							{#if entry.arguments}
								<details>
									<summary class="flex cursor-pointer items-center gap-2">
										<WrenchIcon class="size-3 shrink-0" />
										<span class="min-w-0 flex-1 truncate font-mono"
											>{entry.tool}</span
										>
										{@render time(entry)}
									</summary>
									<div class="mt-2 space-y-1">
										<p>{m.Tool_call_arguments()}</p>
										<CodeDisplay content={entry.arguments} language="json" />
									</div>
								</details>
							{:else}
								<div class="flex items-center gap-2">
									<WrenchIcon class="size-3 shrink-0" />
									<span class="min-w-0 flex-1 truncate font-mono"
										>{entry.tool}</span
									>
									{@render time(entry)}
								</div>
							{/if}
						</div>
					{:else if entry.kind === 'review'}
						<div class="flex items-center gap-2">
							{#if entry.accepted}
								<Badge variant="secondary">
									<CheckIcon />
									{m.Transcript_save_accepted()}
								</Badge>
							{:else}
								<Badge variant="outline">
									<XIcon />
									{m.Transcript_save_refused()}
								</Badge>
							{/if}
							{@render time(entry)}
						</div>
					{:else if entry.kind === 'saved'}
						<div class="space-y-2 rounded-md border p-3 text-sm">
							<div class="flex flex-wrap items-center gap-2">
								<SaveIcon class="size-4 shrink-0" />
								<span class="min-w-0 flex-1 font-medium wrap-anywhere">
									{m.Transcript_saved({ name: entry.name ?? '' })}
								</span>
								{#if entry.replaced}
									<Badge variant="outline">{m.Transcript_replaced()}</Badge>
								{/if}
								{@render time(entry)}
							</div>
							{#if entry.code}
								<CodeDisplay content={entry.code} language="yaml" />
							{/if}
						</div>
					{:else}
						<div class="flex items-center gap-2 text-xs text-muted-foreground italic">
							<span class="flex-1">{m.Transcript_unavailable()}</span>
							{@render time(entry)}
						</div>
					{/if}
				</li>
			{/each}
		</ol>
	{/if}
</div>
