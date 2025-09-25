<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export type RowAfterProps<Response extends object> = {
		Tr: typeof Table.Row;
		Td: typeof Table.Cell;
		record: Response;
	};
</script>

<script lang="ts" generics="Response extends object">
	import type { Snippet } from 'svelte';

	import type { CollectionName } from '@/pocketbase/collections-models';
	import type { CollectionResponses } from '@/pocketbase/types';
	import type { KeyOf } from '@/utils/types';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import {
		RecordDelete,
		RecordEdit,
		RecordSelect,
		RecordShare,
		type RecordAction
	} from '../record-actions';
	import FieldTh from './fieldTh.svelte';

	//

	interface Props {
		records: Response[];
		fields?: KeyOf<Response>[];
		snippets?: Partial<Record<KeyOf<Response>, Snippet<[Response]>>>;
		hide?: Array<RecordAction>;
		header?: Snippet<[{ Th: typeof Table.Head }]>;
		row?: Snippet<[{ Td: typeof Table.Cell; record: Response }]>;
		actions?: Snippet<[{ record: Response }]>;
		rowCellClass?: string;
		rowClass?: string;
		headerClass?: string;
		class?: string;
		rowAfter?: Snippet<[RowAfterProps<Response>]>;
	}

	const {
		records,
		fields = ['id'] as KeyOf<Response>[],
		hide = [],
		header,
		row,
		actions,
		snippets,
		rowCellClass,
		rowClass,
		headerClass,
		class: className,
		rowAfter
	}: Props = $props();

	const hasRightActions = $derived(
		!hide.includes('edit') || !hide.includes('share') || !hide.includes('delete') || actions
	);
</script>

<Table.Root class={className}>
	<Table.Header class={['sticky top-0', headerClass]}>
		<Table.Row class="border-b hover:bg-inherit">
			{#if !hide.includes('select')}
				<Table.Head />
			{/if}

			{#each fields as field}
				<FieldTh {field} />
			{/each}

			{@render header?.({ Th: Table.Head })}

			{#if hasRightActions}
				<Table.Head>
					{m.Actions()}
				</Table.Head>
			{/if}
		</Table.Row>
	</Table.Header>

	<Table.Body>
		{#each records as untypedRecord (untypedRecord)}
			{@const record = untypedRecord as CollectionResponses[CollectionName]}
			<Table.Row
				class={['hover:bg-inherit', 'has-[+tr.hide-previous-border]:border-none', rowClass]}
			>
				{#if !hide.includes('select')}
					<Table.Cell class="py-2">
						<RecordSelect {record} />
					</Table.Cell>
				{/if}

				{#each fields as field}
					{@const snippet = snippets?.[field]}
					<Table.Cell class={rowCellClass}>
						{#if snippet}
							{@render snippet(untypedRecord)}
						{:else}
							{untypedRecord[field]}
						{/if}
					</Table.Cell>
				{/each}

				{@render row?.({ record: untypedRecord, Td: Table.Cell })}

				{#if hasRightActions}
					<Table.Cell class="py-2">
						{@render actions?.({ record: untypedRecord })}

						{#if !hide.includes('edit')}
							<RecordEdit {record}>
								{#snippet button({ triggerAttributes, icon })}
									<IconButton {icon} variant="ghost" {...triggerAttributes} />
								{/snippet}
							</RecordEdit>
						{/if}
						{#if !hide.includes('share')}
							<RecordShare {record}>
								{#snippet button({ triggerAttributes, icon })}
									<IconButton {icon} variant="ghost" {...triggerAttributes} />
								{/snippet}
							</RecordShare>
						{/if}
						{#if !hide.includes('delete')}
							<RecordDelete {record}>
								{#snippet button({ triggerAttributes, icon })}
									<IconButton {icon} variant="ghost" {...triggerAttributes} />
								{/snippet}
							</RecordDelete>
						{/if}
					</Table.Cell>
				{/if}
			</Table.Row>

			{@render rowAfter?.({ Tr: Table.Row, Td: Table.Cell, record: untypedRecord })}
		{/each}
	</Table.Body>
</Table.Root>
