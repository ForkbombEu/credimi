<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { IconComponent } from '@/components/types';

	export interface IndexItem {
		icon?: IconComponent;
		anchor: string;
		label: string;
	}
</script>

<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = {
		sections: IndexItem[];
		title?: string;
	} & HTMLAttributes<HTMLUListElement>;

	let { sections, class: className, title, ...props }: Props = $props();
</script>

<div class={['space-y-4', className]}>
	{#if title}
		<p class="border-b pb-1 text-lg font-semibold">{title}</p>
	{/if}

	<ul class="space-y-4" {...props}>
		{#each sections as section (section.anchor)}
			<li>
				<a href="#{section.anchor}" class="flex items-center gap-2 hover:underline">
					{#if section.icon}
						<section.icon class="size-4 shrink-0" />
					{/if}
					{section.label}
				</a>
			</li>
		{/each}
	</ul>
</div>
