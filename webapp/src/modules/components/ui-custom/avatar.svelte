<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import * as Avatar from '@/components/ui/avatar';
	import type { ComponentProps } from 'svelte';

	export type AvatarProps = ComponentProps<typeof Avatar.Root> & {
		src?: string;
		alt?: string;
		fallback?: string;
		hideIfLoadingError?: boolean;
	};
</script>

<script lang="ts">
	import { m } from '@/i18n';
	import { cn } from '../ui/utils';

	const { src, alt, fallback, hideIfLoadingError = false, ...rest }: AvatarProps = $props();

	//

	let loadingError = $state(false);

	if (src) {
		const tester = new Image();
		tester.src = src;
		tester.onerror = () => {
			loadingError = true;
		};
	}
</script>

{#if !(loadingError && hideIfLoadingError)}
	<Avatar.Root {...rest} class={cn(rest.class, 'overflow-hidden')}>
		{#if src}
			<Avatar.Image {src} alt={alt ?? m.Avatar()} />
		{/if}
		{#if fallback}
			<Avatar.Fallback class="rounded-none text-[80%] font-semibold uppercase">
				{fallback}
			</Avatar.Fallback>
		{/if}
	</Avatar.Root>
{/if}
