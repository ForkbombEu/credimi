<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { LinkWithIcon } from '@/components/types';

	import Button from '@/components/ui-custom/button.svelte';
	import IconComponent from '@/components/ui-custom/icon.svelte';
	import { Badge } from '@/components/ui/badge';

	interface Props {
		link: LinkWithIcon;
		variant?: 'desktop' | 'mobile';
		badge?: string;
	}

	const { link, variant = 'desktop', badge }: Props = $props();
	const { href, title, icon, ...rest } = $derived(link);
</script>

{#if variant === 'desktop'}
	<Button class="group" variant="link" {href} {...rest as any}>
		{#if icon}
			<IconComponent src={icon} />
		{/if}
		{title}
		{@render badgeSnippet()}
	</Button>
{:else}
	<Button variant="ghost" {href} class="group w-full justify-start text-left" {...rest as any}>
		{#if icon}
			<IconComponent src={icon} />
		{/if}
		{title}
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
