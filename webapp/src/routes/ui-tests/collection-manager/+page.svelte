<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager, type FilterGroup } from '@/collections-components/manager';

	const filters: FilterGroup[] = [
		{
			name: 'Group 1',
			id: 'default',
			mode: '&&',
			filters: [
				{
					name: 'Has self relation',
					expression: 'self_relation != null'
				},
				{
					name: 'Self relation has Select B',
					expression: "self_relation.select_field = 'b'"
				}
			]
		}
	];
</script>

<CollectionManager
	collection="z_test_collection"
	{filters}
	queryOptions={{
		expand: ['relation_field'],
		perPage: 6,
		sort: ['self_relation', 'ASC']
	}}
>
	{#snippet top({ Header, Search, Filters })}
		<Header />

		<div class="mt-4 mb-4 flex gap-4">
			<Search containerClass="grow" />
			<Filters modalType="sheet">Filters in sheet</Filters>
			<Filters modalType="popover">Filters in popover</Filters>
		</div>
	{/snippet}

	{#snippet records({ records, Table, Card })}
		<Table {records} fields={['id', 'text_field', 'self_relation', 'select_field']}></Table>

		<div class="mt-4 space-y-2">
			{#each records as record (record.id)}
				<Card {record}>
					{#snippet children({ Title, Description })}
						<Title>{record.text_field}</Title>
						<Description>{record.expand?.relation_field?.email}</Description>
					{/snippet}
				</Card>
			{/each}
		</div>
	{/snippet}
</CollectionManager>
