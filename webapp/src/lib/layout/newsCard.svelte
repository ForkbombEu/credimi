<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CalendarDays, Clock } from '@lucide/svelte';
	import { resolve } from '$app/paths';

	import type { NewsResponse } from '@/pocketbase/types';

	import HTML from '@/components/ui-custom/renderHTML.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { cn } from '@/components/ui/utils';

	type Props = {
		news: NewsResponse;
		class?: string;
	};

	const { news, class: className }: Props = $props();
</script>

<div
	class={cn(
		'border-primary bg-card text-card-foreground ring-primary relative',
		'flex flex-col justify-between gap-4',
		'overflow-visible rounded-lg border p-4 shadow-sm transition-all hover:-translate-y-2 hover:ring-2',
		className
	)}
>
	<a href={resolve("/(public)/news/[news_id]", { news_id: news.id })}>
		<div class="text-muted-foreground mb-3 flex items-center gap-4 text-sm">
			<div class="text-muted-foreground mb-3 flex items-center gap-4 text-sm">
				<div class="flex items-center gap-1">
					<CalendarDays class="h-4 w-4" />
					<time dateTime={news.created}>
						{new Date(news.created).toLocaleDateString('en-US', {
							year: 'numeric',
							month: 'long',
							day: 'numeric'
						})}
					</time>
				</div>
				<div class="flex items-center gap-1">
					<Clock class="h-4 w-4" />
					<span>5 min</span>
				</div>
			</div>
		</div>
		<div class="flex flex-col gap-3">
			<T tag="h3" class="prose block">{news.title}</T>
			<HTML class="prose prose-sm text-primary" content={news.summary} />
		</div>
	</a>
</div>
