<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowUpRight } from '@lucide/svelte';

	import A from '@/components/ui-custom/a.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';

	import type { ChildLink, Item, Layout } from './types';

	import EntityAvatar from './avatar.svelte';
	import EntityChildren from './children.svelte';
	import StackedItems from './stacked-items.svelte';

	//

	type Props = {
		item: Item;
		layout: Layout;
	};

	let { item, layout }: Props = $props();

	const stackableChildren = $derived(item.children?.filter((child) => child.avatar) ?? []);
</script>

{#if layout === 'avatar-only'}
	<IconButton
		href={item.href}
		icon={ArrowUpRight}
		size="sm"
		variant="ghost"
		class="text-primary"
		tooltip={item.name}
		aria-label={item.name}
	/>
{:else if layout === 'logos'}
	{#if item.children?.length}
		<div class="flex flex-wrap items-center gap-1.5">
			{#each item.children as child (child.href)}
				<Tooltip>
					{#if child.avatar}
						<EntityAvatar
							item={{
								key: child.href,
								name: child.label,
								href: child.href,
								avatar: child.avatar
							}}
							link
						/>
					{:else}
						<a
							href={child.href}
							class="rounded-sm text-xs hover:underline focus-visible:outline-2"
						>
							{child.label}
						</a>
					{/if}
					{#snippet content()}
						<p class="text-xs font-semibold">{child.label}</p>
					{/snippet}
				</Tooltip>
			{/each}
		</div>
	{/if}
{:else if layout === 'links-only'}
	<A href={item.href} class="block max-w-[30ch] truncate text-xs">
		{item.name}
	</A>
{:else if layout === 'compact'}
	<div class="flex items-center gap-2">
		{#if stackableChildren.length}
			{#snippet stackedChild({ item: child }: { item: ChildLink })}
				<EntityAvatar
					item={{
						key: child.href,
						name: child.label,
						href: child.href,
						avatar: child.avatar
					}}
					link
				/>
			{/snippet}
			<StackedItems
				items={stackableChildren}
				getKey={(child) => child.href}
				item={stackedChild}
			/>
		{:else if item.avatar}
			<EntityAvatar {item} link />
		{/if}
		<div class="flex min-w-0 flex-col text-xs">
			<span class="flex items-center gap-1">
				<A href={item.href} class="font-semibold whitespace-nowrap">
					{item.name}
				</A>
				{#if item.children && item.children.length > 1}
					<span
						class="inline-flex min-w-4 items-center justify-center rounded-full bg-secondary px-1 text-[9px] leading-4 font-semibold text-secondary-foreground tabular-nums"
						aria-label="{item.children.length} {item.kind?.labels.plural ?? 'items'}"
					>
						{item.children.length}
					</span>
				{/if}
			</span>
			{#if item.caption}
				<p class="font-mono text-[10px] text-muted-foreground">
					{item.caption}
				</p>
			{/if}
		</div>
	</div>
{:else}
	<div class="flex items-start gap-2">
		{#if item.avatar}
			<EntityAvatar {item} link />
		{/if}
		<div class="flex flex-col">
			<A href={item.href} class="text-xs font-bold">
				{item.name}
			</A>
			{#if item.caption}
				<p class="text-xs text-muted-foreground">
					{item.caption}
				</p>
			{/if}
			{#if item.children?.length}
				<EntityChildren children={item.children} />
			{/if}
		</div>
	</div>
{/if}
