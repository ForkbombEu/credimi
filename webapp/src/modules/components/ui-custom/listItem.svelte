<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { X } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';

	import T from './t.svelte';

	interface Props {
		label?: string | undefined;
		hideButton?: boolean;
		class?: string;
		contentClass?: string;
		children?: Snippet;
		onclick?: () => void;
	}

	let {
		label = undefined,
		hideButton = false,
		class: className = '',
		contentClass = '',
		children,
		onclick = () => {}
	}: Props = $props();
</script>

<div class="flex items-center justify-between gap-6 px-1 py-1 pl-3 {className}">
	<div class="w-0 grow overflow-hidden {contentClass}">
		{#if children}{@render children()}{:else if label}
			<T tag="p">
				{label}
			</T>
		{/if}
	</div>

	{#if !hideButton}
		<Button variant="ghost" size="icon" class="size-8" {onclick}>
			<Icon src={X} size={20} />
		</Button>
	{/if}
</div>
