<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ArrowRightIcon } from 'lucide-svelte';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		avatar?: string;
		subtitle?: string;
		title: string;
		onClick?: () => void;
		right?: Snippet;
	};

	let { avatar, title, onClick, subtitle, right }: Props = $props();
	const isInteractive = $derived(onClick !== undefined);

	const classes = 'flex w-full items-center gap-3 rounded-md  p-2 text-left';
</script>

{#if onClick}
	<button class={['bg-card border hover:ring', classes]} onclick={() => onClick()}>
		{@render content?.()}
	</button>
{:else}
	<div class={['bg-slate-100', classes]}>
		{@render content?.()}
	</div>
{/if}

{#snippet content()}
	{#if avatar}
		<Avatar src={avatar} fallback={title} class="shrink-0 rounded-md border" />
	{/if}

	<div class="grow">
		<T class="text-sm font-semibold">{title}</T>
		{#if subtitle}
			<T class="text-muted-foreground text-xs">{subtitle}</T>
		{/if}
	</div>

	<div class="shrink-0">
		{@render right?.()}
		{#if isInteractive}
			<ArrowRightIcon class="text-muted-foreground size-4 shrink-0" />
		{/if}
	</div>
{/snippet}
