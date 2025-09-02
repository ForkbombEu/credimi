<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ContentPage } from '$lib/content/types';

	import { getTagTranslation } from '$lib/content/tags-i18n';

	import T from '@/components/ui-custom/t.svelte';
	import { localizeHref } from '@/i18n';

	const { attributes, slug }: ContentPage = $props();
	const { title, date, description, tags } = attributes;
</script>

<a
	href={localizeHref(`/${slug}`)}
	class="border-primary bg-card text-card-foreground ring-primary flex flex-col justify-between gap-4 rounded-lg border p-6 shadow-sm transition-all hover:-translate-y-2 hover:ring-2"
>
	<div>
		<T tag="h4" class="text-balance font-bold">
			{title}
		</T>

		<T class="text-muted-foreground text-sm">
			{new Date(date).toLocaleDateString(undefined, {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			})}
		</T>
	</div>

	{#if description}
		<T tag="p" class="text-primary line-clamp-3">
			{description}
		</T>
	{/if}

	<div class="tags flex flex-wrap items-center gap-1.5">
		{#each tags as tag}
			<span class="text-primary border-primary rounded-lg border px-1.5 text-xs">
				{getTagTranslation(tag)}
			</span>
		{/each}
	</div>
</a>
