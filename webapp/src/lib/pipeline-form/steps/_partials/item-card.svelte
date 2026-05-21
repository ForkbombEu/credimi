<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import { ArrowRightIcon, XIcon } from '@lucide/svelte';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { cn } from '@/components/ui/utils';

	type Props = {
		avatar?: string;
		subtitle?: string;
		title: string;
		onClick?: (e: MouseEvent) => void;
		onDiscard?: () => void;
		right?: Snippet;
		class?: ClassValue;
		tooltip?: string;
		beforeContent?: Snippet;
		afterContent?: Snippet;
		titleRight?: Snippet;
		hideArrow?: boolean;
	};

	let {
		avatar,
		title,
		onClick,
		onDiscard,
		subtitle,
		right,
		class: className,
		tooltip,
		beforeContent,
		afterContent,
		titleRight,
		hideArrow = false
	}: Props = $props();
	const isInteractive = $derived(onClick !== undefined);

	const classes = $derived(
		cn('gap-3 rounded-md border border-slate-200 p-2 text-left w-full', className)
	);
</script>

<Tooltip disabled={!tooltip}>
	{#snippet child({ props })}
		{#if onClick}
			<button class={['bg-card hover:ring', classes]} onclick={(e) => onClick(e)} {...props}>
				{@render itemContent()}
			</button>
		{:else}
			<div class={['bg-slate-100', classes]} {...props}>
				{@render itemContent()}
			</div>
		{/if}
	{/snippet}

	{#snippet content()}
		{tooltip}
	{/snippet}
</Tooltip>

{#snippet itemContent()}
	<div class="flex items-center gap-3">
		<div class="grow">
			{@render beforeContent?.()}

			<div class="flex items-center gap-2">
				{#if avatar}
					<Avatar src={avatar} fallback={title} class="shrink-0 rounded-md border" />
				{/if}
				<div>
					<T class="text-sm font-semibold">
						<span>
							{title}
						</span>
						{@render titleRight?.()}
					</T>
					{#if subtitle}
						<T class="text-xs text-muted-foreground">{subtitle}</T>
					{/if}
				</div>
			</div>

			{@render afterContent?.()}
		</div>

		<div class="flex shrink-0 items-center gap-1">
			{@render right?.()}
			{#if onDiscard}
				<IconButton
					icon={XIcon}
					variant="ghost"
					size="xs"
					class="hover:bg-slate-200"
					onclick={() => onDiscard()}
				/>
			{/if}
			{#if isInteractive && !hideArrow}
				<ArrowRightIcon class="size-4 shrink-0 text-muted-foreground" />
			{/if}
		</div>
	</div>
{/snippet}
