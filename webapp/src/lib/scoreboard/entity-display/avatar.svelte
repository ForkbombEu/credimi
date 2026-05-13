<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { resolve } from '$app/paths';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import { localizeHref } from '@/i18n';

	import type { Item } from './types';

	//

	type Props = {
		item: Item;
		link?: boolean;
	};

	let { item, link = false }: Props = $props();
</script>

{#if item.avatar}
	{#if link}
		<a
			href={resolve(localizeHref(item.href) as '/')}
			class="w-fit shrink-0 rounded-sm hover:ring-2 hover:ring-primary"
		>
			{@render content()}
		</a>
	{:else}
		{@render content()}
	{/if}
{/if}

{#snippet content()}
	{#if item.avatar}
		<Avatar
			src={item.avatar.src}
			fallback={item.avatar.fallback}
			alt={item.avatar.alt}
			class="size-8 rounded-sm border bg-muted uppercase"
		/>
	{/if}
{/snippet}
