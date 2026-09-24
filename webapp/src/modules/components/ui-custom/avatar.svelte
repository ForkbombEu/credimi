<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { HTMLAttributes } from 'svelte/elements';

	export type AvatarProps = Omit<HTMLAttributes<HTMLDivElement>, 'children'> & {
		ref?: HTMLElement | null;
		src?: string;
		alt?: string;
		fallback?: string;
		hideIfLoadingError?: boolean;
		fallbackLength?: number | null | undefined;
	};
</script>

<script lang="ts">
	import { m } from '@/i18n';

	import { cn } from '../ui/utils';

	let {
		src,
		alt,
		fallback,
		hideIfLoadingError = false,
		fallbackLength = 2,
		class: className,
		ref = $bindable(null),
		...rest
	}: AvatarProps = $props();

	//

	let imageFailed = $state(false);

	$effect(() => {
		void src;
		imageFailed = false;
	});
</script>

{#if !(imageFailed && hideIfLoadingError)}
	<div
		bind:this={ref}
		data-slot="avatar"
		class={cn('relative flex size-8 shrink-0 overflow-hidden rounded-full', className)}
		{...rest}
	>
		{#if src && !imageFailed}
			<img
				{src}
				alt={alt ?? m.Avatar()}
				class="size-full object-cover"
				loading="lazy"
				decoding="async"
				onerror={() => {
					imageFailed = true;
				}}
			/>
		{/if}
		{#if fallback && (!src || imageFailed)}
			<span
				class="flex size-full items-center justify-center rounded-none text-[80%] font-semibold uppercase"
			>
				{fallbackLength ? fallback.slice(0, fallbackLength) : fallback}
			</span>
		{/if}
	</div>
{/if}
