<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { FileCogIcon, ImageIcon, VideoIcon } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';

	//

	type IconType = 'image' | 'video' | 'file';

	type Props = {
		image?: string;
		href: string;
		icon: IconComponent | IconType;
	};

	let { image, href, icon }: Props = $props();

	const map: Record<IconType, IconComponent> = {
		image: ImageIcon,
		video: VideoIcon,
		file: FileCogIcon
	};

	const iconComponent = $derived<IconComponent>(typeof icon === 'string' ? map[icon] : icon);
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -->
<a
	{href}
	target="_blank"
	class="relative size-10 shrink-0 overflow-hidden rounded-md border border-slate-300 hover:cursor-pointer hover:ring-2"
>
	{#if image}
		<img src={image} alt="Media" class="size-10 shrink-0" />
	{/if}
	<div class="absolute inset-0 flex items-center justify-center bg-black/30">
		<Icon src={iconComponent} class="size-4  text-white" />
	</div>
</a>
