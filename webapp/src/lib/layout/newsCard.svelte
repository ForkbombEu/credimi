<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import HTML from '@/components/ui-custom/renderHTML.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { cn } from '@/components/ui/utils';
	import { m } from '@/i18n';
	import type { NewsResponse } from '@/pocketbase/types';
	import { CalendarDays, Clock } from "lucide-svelte"

	type Props = {
		news: NewsResponse;
		class?: string;
	};

	const { news, class: className }: Props = $props();
</script>

<div class={cn('flex flex-col rounded-lg border bg-card text-card-foreground shadow-sm hover:shadow-xl transition-shadow duration-200 p-4', className)}>
	<a href="/news/{news.id}">
	<div class="flex items-center gap-4 text-sm text-muted-foreground mb-3" >
		<div class="flex items-center gap-4 text-sm text-muted-foreground mb-3">
			<div class="flex items-center gap-1">
				<CalendarDays class="h-4 w-4" />
				<time dateTime={news.created}>
					{new Date(news.created).toLocaleDateString("en-US", {
					year: "numeric",
					month: "long",
					day: "numeric",
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
