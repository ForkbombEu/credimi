<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import A from '@/components/ui-custom/a.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';

	import type { Item, Layout } from './types';

	import EntityAvatar from './avatar.svelte';
	import EntityChildren from './children.svelte';

	//

	type Props = {
		item: Item;
		layout: Layout;
	};

	let { item, layout }: Props = $props();
</script>

{#if layout === 'avatar-only'}
	<EntityAvatar {item} link />
{:else if layout === 'links-only'}
	<A href={item.href} class="block max-w-[30ch] truncate text-xs">
		{item.name}
	</A>
{:else if layout === 'compact'}
	<div class="flex items-center gap-2">
		{#if item.avatar}
			<EntityAvatar {item} link />
		{/if}
		<div class="flex flex-col text-xs">
			<A href={item.href} class="max-w-[15ch] truncate">
				{item.name}
			</A>
			{#if item.kind}
				<p class={[item.kind.classes.text]}>
					<item.kind.icon class="inline-block size-3 -translate-y-px" />
					{item.kind.labels.singular}
					{#if item.children?.length}
						<Tooltip>
							<span
								class={[
									'inline-flex min-w-5 items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] leading-none tabular-nums transition-colors',
									item.kind.classes.text,
									'bg-current/15 hover:bg-current/35'
								]}
							>
								{item.children.length}
							</span>

							{#snippet content()}
								<EntityChildren
									children={item.children!}
									linkClass="text-white hover:underline"
								/>
							{/snippet}
						</Tooltip>
					{/if}
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
