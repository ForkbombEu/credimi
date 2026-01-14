<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as Select from '@/components/ui/select';

	type SelectItem = {
		value: string;
		label: string;
		disabled?: boolean;
	};

	type Props = {
		items: SelectItem[];
		value?: string;
		placeholder?: string;
	};

	let { items, value = $bindable(), placeholder }: Props = $props();

	const triggerContent = $derived(
		items.find((f) => f.value === value)?.label ?? placeholder ?? 'Select a value'
	);
</script>

<Select.Root type="single" bind:value>
	<Select.Trigger>
		{triggerContent}
	</Select.Trigger>
	<Select.Content>
		<Select.Group>
			{#each items as item (item.value)}
				<Select.Item value={item.value} label={item.label} disabled={item.disabled}>
					{item.label}
				</Select.Item>
			{/each}
		</Select.Group>
	</Select.Content>
</Select.Root>
