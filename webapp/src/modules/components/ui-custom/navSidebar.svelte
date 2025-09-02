<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import { cubicInOut } from 'svelte/easing';
	import { crossfade } from 'svelte/transition';

	import { Button } from '@/components/ui/button/index.js';
	import { cn } from '@/components/ui/utils.js';

	import type { LinkWithIcon } from '../types';

	import Icon from './icon.svelte';
	import T from './t.svelte';

	type Props = {
		class?: string;
		links: LinkWithIcon[];
		title?: string;
	};

	const { class: className, links, title }: Props = $props();

	const [send, receive] = crossfade({
		duration: 250,
		easing: cubicInOut
	});
</script>

<nav class={cn('flex flex-col', className)}>
	{#if title}
		<T tag="small" class="mb-4 text-xs font-semibold uppercase">{title}</T>
	{/if}

	{#each links as { href, title, icon }}
		{@const isActive = page.url.pathname === href}

		<Button
			{href}
			variant="ghost"
			class={cn(
				!isActive && 'hover:underline',
				'relative justify-start hover:bg-transparent'
			)}
			data-sveltekit-noscroll
			size="sm"
		>
			{#if isActive}
				<div
					class="absolute inset-0 rounded-md bg-background"
					in:send={{ key: 'active-sidebar-tab' }}
					out:receive={{ key: 'active-sidebar-tab' }}
				></div>
			{/if}

			<div class="relative flex items-center">
				{#if icon}
					<Icon src={icon} mr />
				{/if}
				{title}
			</div>
		</Button>
	{/each}
</nav>
