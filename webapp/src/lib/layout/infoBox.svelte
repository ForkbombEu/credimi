<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		label?: string;
		value?: string;
		children?: Snippet;
	};

	let { label, children, value }: Props = $props();
</script>

{#if value || children}
	<div class="space-y-1">
		{#if label}
			<T class="text-sm">{label}:</T>
		{/if}

		<div class="w-fit rounded-sm border border-slate-400 bg-white px-2 py-1">
			{#if value}
				<T class="prose-lg whitespace-pre-wrap">{value}</T>
			{:else if children}
				{@render children()}
			{/if}
		</div>
	</div>
{/if}
