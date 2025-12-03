<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import type { ControlAttrs } from 'formsnap';

	import { nanoid } from 'nanoid';
	import { onMount } from 'svelte';

	import * as Select from '@/components/ui/select';

	import type { SelectOption } from './utils';

	//

	type Props = {
		items: SelectOption<T>[];
		value: T | undefined | null;
		placeholder?: string;
		controlAttrs?: ControlAttrs;
	};

	let { items, value = $bindable(), placeholder, controlAttrs }: Props = $props();

	//

	const itemsWithId = $derived(items.map((item) => ({ ...item, id: nanoid(4) })));

	let selectedId = $state<string>();
	const selectedItem = $derived(itemsWithId.find((item) => item.id === selectedId));

	onMount(() => {
		if (!value) return;
		selectedId = itemsWithId.find((item) => item.value === value)?.id;
	});

	$effect(() => {
		const newValue = itemsWithId.find((item) => item.id === selectedId)?.value;
		if (newValue !== value) value = newValue;
	});
</script>

<Select.Root type="single" bind:value={selectedId}>
	<Select.Trigger {...controlAttrs}>
		{#if selectedItem}
			{selectedItem.label}
		{:else}
			{placeholder}
		{/if}
	</Select.Trigger>
	<Select.Content>
		{#each itemsWithId as item (item.id)}
			<Select.Item value={item.id}>{item.label}</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>
