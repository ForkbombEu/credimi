<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Button from '@/components/ui-custom/button.svelte';
	import IconComponent from '@/components/ui-custom/icon.svelte';
	import type { LinkWithIcon } from '@/components/types';
	import { Badge } from '@/components/ui/badge';

	interface Props {
		link: LinkWithIcon;
		variant?: 'desktop' | 'mobile';
		badge?: string;
	}

	const { link, variant = 'desktop', badge }: Props = $props();
</script>

{#if variant === 'desktop'}
	<Button class="group" variant="link" href={link.href}>
		{#if link.icon}
			<IconComponent src={link.icon} />
		{/if}
		{link.title}
		{@render badgeSnippet()}
	</Button>
{:else}
	<Button variant="ghost" href={link.href} class="group w-full justify-start text-left ">
		{#if link.icon}
			<IconComponent src={link.icon} />
		{/if}
		{link.title}
		{@render badgeSnippet()}
	</Button>
{/if}

{#snippet badgeSnippet()}
	{#if badge}
		<Badge
			variant="outline"
			class="border-primary text-primary text-xs group-hover:no-underline"
		>
			{badge}
		</Badge>
	{/if}
{/snippet}
