<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script
	lang="ts"
	generics="C extends CollectionName, E extends PocketbaseQueryExpandOption<C> = never"
>
	import type { CollectionName } from '@/pocketbase/collections-models';
	import type { RecordIdString } from '@/pocketbase/types';

	import SelectInput, { type SelectItem } from '@/components/ui-custom/selectInput.svelte';
	import {
		type PocketbaseQueryResponse,
		type PocketbaseQueryExpandOption,
		type PocketbaseQueryOptions,
		PocketbaseQueryAgent
	} from '@/pocketbase/query';
	import { setupComponentPocketbaseSubscriptions } from '@/pocketbase/subscriptions';

	import type { CollectionInputProps } from './types';

	import { createRecordDisplay } from './utils';

	//

	type Props = CollectionInputProps<C, E>;

	let {
		collection,
		queryOptions = {},
		disabled = false,
		placeholder,
		clearValueOnSelect = false,
		onSelect = () => {},
		displayFields,
		displayFn,
		controlAttrs
	}: Props = $props();

	//

	type Record = PocketbaseQueryResponse<C, E>;

	let records = $state<Record[]>([]);
	let recordId = $state<RecordIdString | undefined>();
	const selectedRecord = $derived(records.find((r) => r.id == recordId));

	const presentRecord = $derived(function (record: Record) {
		return createRecordDisplay(record, displayFields, displayFn);
	});

	const selectItems: SelectItem[] = $derived(
		records.map((r) => ({
			value: r.id,
			label: presentRecord(r)
		}))
	);

	//

	const loadRecords = $derived(function () {
		const runners = new PocketbaseQueryAgent({ collection, ...queryOptions });
		runners.getFullList().then((res) => (records = res));
	});

	$effect(() => {
		loadRecords();
	});

	setupComponentPocketbaseSubscriptions({
		collection,
		callback: () => loadRecords(),
		expandOption: queryOptions.expand,
		other: ['authorizations']
	});

	//

	$effect(() => {
		if (!selectedRecord) return;
		onSelect($state.snapshot(selectedRecord) as Record);
		if (clearValueOnSelect) recordId = undefined;
	});
</script>

<SelectInput
	type="single"
	items={selectItems}
	bind:value={recordId}
	{placeholder}
	{controlAttrs}
	{disabled}
>
	{#snippet trigger()}
		{#if selectedRecord}
			{presentRecord(selectedRecord)}
		{/if}
	{/snippet}
</SelectInput>
