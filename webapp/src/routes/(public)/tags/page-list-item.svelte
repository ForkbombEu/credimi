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
	class="flex flex-col justify-between gap-4 rounded-lg border border-primary bg-card p-6 text-card-foreground shadow-sm ring-primary transition-all hover:-translate-y-2 hover:ring-2"
>
	<div>
		<T tag="h4" class="text-balance font-bold">
			{title}
		</T>

		<T class="text-sm text-muted-foreground">
			{new Date(date).toLocaleDateString(undefined, {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			})}
		</T>
	</div>

	{#if description}
		<T tag="p" class="line-clamp-3 text-primary">
			{description}
		</T>
	{/if}

	<div class="tags flex flex-wrap items-center gap-1.5">
		{#each tags as tag}
			<span class="rounded-lg border border-primary px-1.5 text-xs text-primary">
				{getTagTranslation(tag)}
			</span>
		{/each}
	</div>
</a>
