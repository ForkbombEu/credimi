<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { HTMLAnchorAttributes } from 'svelte/elements';

	import { ExternalLink } from 'lucide-svelte';

	import type { IconComponent } from '@/components/types';

	import { cn } from '../ui/utils';

	//

	type Props = HTMLAnchorAttributes & {
		text: string;
		icon?: IconComponent;
		background?: string;
		foreground?: string;
	};

	let {
		href,
		text,
		icon,
		background = 'var(--secondary)',
		foreground = 'var(--secondary-foreground)',
		class: className,
		...rest
	}: Props = $props();
</script>

<a
	{href}
	target="_blank"
	rel="noopener noreferrer"
	class={cn(
		'inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors hover:brightness-90',
		className
	)}
	style="background-color: hsl({background}); color: hsl({foreground});"
	{...rest}
>
	{#if icon}
		{@const IconComponent = icon}
		<IconComponent class="h-3 w-3" />
	{/if}
	{text}
	<ExternalLink class="h-3 w-3" />
</a>
