<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { workflowStatuses, WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { TemporalI18nProvider, type WorkflowStatusType } from '$lib/temporal';

	import T from '@/components/ui-custom/t.svelte';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n';
	import { ensureArray } from '@/utils/other';

	//

	type Props = {
		value?: WorkflowStatusType[];
		onValueChange?: (value: WorkflowStatusType[]) => void;
	};

	let { value = $bindable(), onValueChange }: Props = $props();
</script>

<TemporalI18nProvider>
	<div class="flex flex-col gap-2">
		{#each workflowStatuses as status}
			{#if status}
				<Label class="flex items-center gap-2">
					<Checkbox
						value={status}
						checked={value?.includes(status)}
						onCheckedChange={(checked) => {
							const v = ensureArray(value);
							if (checked) onValueChange?.([...v, status]);
							else onValueChange?.([...v.filter((v) => v !== status)]);
						}}
					/>
					<WorkflowStatus {status} />
				</Label>
			{/if}
		{/each}
	</div>
</TemporalI18nProvider>
