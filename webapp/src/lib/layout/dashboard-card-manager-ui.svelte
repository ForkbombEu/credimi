<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import type { BaseSystemFields } from '@/pocketbase/types';
</script>

<script lang="ts" generics="R extends BaseSystemFields">
	import type { Snippet } from 'svelte';

	import type { StringKey } from '@/utils/types';

	import DashboardCardManagerItem from './dashboard-card-manager-item.svelte';

	//

	type Props = {
		nameField: StringKey<R>;
		fallbackNameField?: StringKey<R>;
		publicUrl?: (record: R) => string;
		records: R[];
		actions?: Snippet<[{ record: R }]>;
		hideClone?: boolean;
		path: (record: R) => string[];
	};

	let {
		nameField,
		fallbackNameField,
		publicUrl,
		records,
		actions: actionsSnippet,
		hideClone,
		path
	}: Props = $props();
</script>

<div>
	<ul class="space-y-2">
		{#each records as record (record.id)}
			<DashboardCardManagerItem
				{record}
				{nameField}
				{fallbackNameField}
				publicUrl={publicUrl?.(record)}
				{hideClone}
				path={path(record)}
			>
				{#snippet actions()}
					{@render actionsSnippet?.({ record })}
				{/snippet}
			</DashboardCardManagerItem>
		{/each}
	</ul>
</div>
