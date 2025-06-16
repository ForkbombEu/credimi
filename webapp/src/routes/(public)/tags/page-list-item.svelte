<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ContentPage } from '$lib/content/types';
	import T from '@/components/ui-custom/t.svelte';

	const { attributes, slug }: ContentPage = $props();
	const { title, date, description, tags } = attributes;
</script>

<div
	class="border-primary bg-card text-card-foreground ring-primary flex flex-col justify-between gap-2 rounded-lg border p-6 shadow-sm transition-all hover:-translate-y-2 hover:ring-2"
>
	<a href={`/${slug}`} class="!cursor-pointer">
		<T tag="h4" class="text-balance !font-bold">
			{title}
		</T>
	</a>

	<T>
		{new Date(date).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		})}
	</T>

	{#if description}
		<T
			tag="p"
			class="text-primary my-4 overflow-hidden text-balance [-webkit-box-orient:vertical] [-webkit-line-clamp:3] [display:-webkit-box]"
		>
			{description}
		</T>
	{/if}

	<div class="tags flex flex-wrap items-center gap-2">
		<T tag="small" class="text-balance !font-normal">
			{'Tags:'}
		</T>
		{#each tags as tag}
			<a
				href={`/tags?search=${encodeURIComponent(tag)}`}
				class="text-primary border-primary !cursor-pointer rounded-lg border px-2"
			>
				{tag}
			</a>
		{/each}
	</div>
</div>
