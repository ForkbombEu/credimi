<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import { FileCogIcon, FileIcon, ImageIcon, VideoIcon } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';
	import { cn } from '@/components/ui/utils';

	//

	type IconType = 'image' | 'video' | 'file' | 'document';

	type Props = {
		image?: string;
		href?: string;
		onClick?: () => void;
		icon: IconComponent | IconType;
		class?: ClassValue;
	};

	let { image, href, onClick, icon, class: className, ...rest }: Props = $props();

	const map: Record<IconType, IconComponent> = {
		image: ImageIcon,
		video: VideoIcon,
		file: FileCogIcon,
		document: FileIcon
	};

	const iconComponent = $derived<IconComponent>(typeof icon === 'string' ? map[icon] : icon);

	const classes = $derived(
		cn([
			'relative size-10 shrink-0 overflow-hidden rounded-md border border-slate-300 hover:cursor-pointer hover:ring-2',
			className
		])
	);
</script>

{#if href}
	<a {href} target="_blank" class={classes} {...rest}>
		{@render content()}
	</a>
{:else}
	<button class={classes} onclick={onClick} {...rest}>
		{@render content()}
	</button>
{/if}

{#snippet content()}
	{#if image}
		<img src={image} alt="Media" class="size-10 shrink-0" />
	{/if}
	<div class="absolute inset-0 flex items-center justify-center bg-black/30">
		<Icon src={iconComponent} class="size-4  text-white" />
	</div>
{/snippet}
