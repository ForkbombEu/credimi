<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { CheckIcon, Copy, ExternalLink } from '@lucide/svelte';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		label?: string;
		value?: string;
		url?: string;
		copyable?: boolean;
		children?: Snippet;
	};

	let { label, children, value, url, copyable = false }: Props = $props();

	let copied = $state(false);

	const handleCopy = () => {
		if (!url && !value) return;
		navigator.clipboard.writeText(url || value || '');
		copied = true;
		setTimeout(() => {
			copied = false;
		}, 1000);
	};
</script>

{#if value || children || url}
	<div class="space-y-1">
		{#if label}
			<div class="flex items-center gap-1 text-sm">
				{#if url}
					<T>{label}</T>
					<ExternalLink class="h-3 w-3 text-gray-500" />
				{:else}
					<T>{label}:</T>
				{/if}
			</div>
		{/if}

		<div
			class="w-fit rounded-sm border border-slate-400 bg-white px-2 py-1 {copyable
				? 'pr-1'
				: ''}"
		>
			<div class="flex min-h-[1.5rem] items-center gap-2">
				<div class="flex-1">
					{#if url}
						<a
							href={url}
							class="text-primary hover:underline"
							target="_blank"
							rel="noopener noreferrer"
						>
							<T class="prose-lg whitespace-pre-wrap">{url}</T>
						</a>
					{:else if value}
						<T class="prose-lg whitespace-pre-wrap">{value}</T>
					{:else if children}
						<div class="prose-lg">
							{@render children()}
						</div>
					{/if}
				</div>
				{#if copyable && (url || value)}
					{#if !copied}
						<button
							type="button"
							onclick={handleCopy}
							class="flex-shrink-0 rounded p-1 transition-colors hover:bg-gray-200"
							title="Copy {label || 'text'}"
						>
							<Copy class="size-3 text-gray-500" />
						</button>
					{:else}
						<div class="p-1">
							<CheckIcon class="size-3 text-green-500" />
						</div>
					{/if}
				{/if}
			</div>
		</div>
	</div>
{/if}
