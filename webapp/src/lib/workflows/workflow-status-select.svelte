<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends 'multiple' | 'single'">
	import { workflowStatuses, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import * as Select from '@/components/ui/select/index.js';
	import { TemporalI18nProvider, type WorkflowStatusType } from '$lib/temporal';
	import { ensureArray } from '@/utils/other';
	import { m } from '@/i18n';

	type Props = {
		type: T;
		value?: (T extends 'multiple' ? WorkflowStatusType[] : WorkflowStatusType) | undefined;
	};

	let { type, value = $bindable() }: Props = $props();
	const maxDisplayed = 2;
</script>

<TemporalI18nProvider>
	<!-- `any` is used to avoid type errors when binding the value -->
	<Select.Root {type} bind:value={value as any}>
		<Select.Trigger class="w-fit gap-2">
			{#each ensureArray(value).slice(0, maxDisplayed) as status}
				{#if status}
					<WorkflowStatus {status} />
				{/if}
			{:else}
				{m.Select_a_value()}
			{/each}
			{#if ensureArray(value).length > maxDisplayed}
				<span
					class=" flex items-center gap-1 whitespace-nowrap rounded-sm bg-gray-100 px-1 py-0.5 font-medium text-black"
				>
					+{ensureArray(value).length - maxDisplayed}
				</span>
			{/if}
		</Select.Trigger>
		<Select.Content>
			{#each workflowStatuses as status}
				{#if status}
					<Select.Item value={status}>
						<WorkflowStatus {status} />
					</Select.Item>
				{/if}
			{/each}
		</Select.Content>
	</Select.Root>
</TemporalI18nProvider>
