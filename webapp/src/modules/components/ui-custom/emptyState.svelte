<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { FolderOpen } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';
	import { cn } from '@/components/ui/utils';

	import T from './t.svelte';

	interface Props {
		title: string;
		description?: string | undefined;
		icon?: IconComponent;
		bottom?: import('svelte').Snippet;
		className?: string;
	}

	let { title, description = undefined, icon = FolderOpen, bottom, className }: Props = $props();
</script>

<div
	class={cn(
		'flex flex-col items-center justify-center rounded-lg border p-10 text-center text-muted-foreground',
		className
	)}
>
	<div class="mb-6 text-primary/40">
		<Icon src={icon} size={70} />
	</div>

	<!-- Not a heading: empty states sit at any depth and would break the page outline. -->
	{#if title}
		<p class="text-xl font-semibold tracking-tight text-slate-700">{title}</p>
	{/if}

	{#if description}
		<T>{description}</T>
	{/if}

	{@render bottom?.()}
</div>
