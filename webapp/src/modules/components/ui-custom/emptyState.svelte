<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { FolderOpen } from 'lucide-svelte';

	import type { IconComponent } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';

	import T from './t.svelte';

	interface Props {
		title: string;
		description?: string | undefined;
		icon?: IconComponent;
		bottom?: import('svelte').Snippet;
		className?: string;
	}

	let { title, description = undefined, icon = FolderOpen, bottom, className }: Props = $props();
</script>

<div
	class={[
		'flex flex-col items-center justify-center rounded-lg border p-10 text-center text-primary/40',
		className
	]}
>
	<div class="mb-6 opacity-50">
		<Icon src={icon} size={70} />
	</div>

	{#if title}
		<T tag="h4">
			{title}
		</T>
	{/if}

	{#if description}
		<T>{description}</T>
	{/if}

	{@render bottom?.()}
</div>
