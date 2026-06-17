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

	import { showPipelineFormError } from '../../errors.js';

	type Props = {
		avatar?: string;
		subtitle?: string;
		title: string;
		onClick?: (e: MouseEvent) => void | Promise<void>;
		onDiscard?: () => void;
		right?: Snippet;
		class?: ClassValue;
		tooltip?: string;
		beforeContent?: Snippet;
		afterContent?: Snippet;
		titleRight?: Snippet;
		hideArrow?: boolean;
		disabled?: boolean;
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
		hideArrow = false,
		disabled = false
	}: Props = $props();
	const isInteractive = $derived(onClick !== undefined && !disabled);

	const classes = $derived(
		cn('gap-3 rounded-md border border-slate-200 p-2 text-left w-full', className)
	);

	async function handleClick(e: MouseEvent) {
		try {
			await onClick?.(e);
		} catch (error) {
			showPipelineFormError(error);
		}
	}
</script>

<Tooltip disabled={!tooltip}>
	{#snippet child({ props })}
		{#if disabled}
			<button
				{disabled}
				class={[
					'cursor-not-allowed bg-card opacity-60 disabled:cursor-not-allowed',
					classes
				]}
				{...props}
			>
				{@render itemContent()}
			</button>
		{:else if onClick}
			<button class={['bg-card hover:ring', classes]} onclick={handleClick} {...props}>
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
	<div class="flex min-w-0 items-center gap-3">
		<div class="w-0 min-w-0 grow overflow-hidden">
			{@render beforeContent?.()}

			<div class="flex min-w-0 items-center gap-2">
				{#if avatar}
					<Avatar src={avatar} fallback={title} class="shrink-0 rounded-md border" />
				{/if}
				<div class="min-w-0 flex-1 overflow-hidden">
					<T class="flex min-w-0 items-center gap-1 text-sm font-semibold">
						<span class="min-w-0 truncate" {title}>
							{title}
						</span>
						{#if titleRight}
							<span class="shrink-0">
								{@render titleRight()}
							</span>
						{/if}
					</T>
					{#if subtitle}
						<T class="truncate text-xs text-muted-foreground">
							<span title={subtitle}>{subtitle}</span>
						</T>
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
