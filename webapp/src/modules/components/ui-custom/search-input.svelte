<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';
	import type { HTMLInputAttributes } from 'svelte/elements';

	import { SearchIcon, XIcon } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Input from '@/components/ui/input/input.svelte';
	import { cn } from '@/components/ui/utils';
	import { m } from '@/i18n';

	//

	type Props = Omit<HTMLInputAttributes, 'type' | 'value' | 'files'> & {
		value?: string;
		class?: ClassValue;
		inputClass?: ClassValue;
		onclear?: () => void;
	};

	let {
		value = $bindable(''),
		class: className,
		inputClass,
		onclear,
		placeholder = m.Search(),
		...restProps
	}: Props = $props();

	function clear() {
		value = '';
		onclear?.();
	}
</script>

<div class={cn('relative', className)}>
	<SearchIcon
		class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 opacity-50"
		aria-hidden="true"
	/>
	<Input
		type="search"
		class={cn('pr-9 pl-9 text-sm', inputClass)}
		bind:value
		{placeholder}
		{...restProps}
	/>
	{#if value}
		<div class="absolute top-0 right-0 p-1">
			<IconButton
				icon={XIcon}
				variant="ghost"
				size="sm"
				aria-label={m.Clear()}
				onclick={clear}
			/>
		</div>
	{/if}
</div>
