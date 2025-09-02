<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import type { CollectionName } from '@/pocketbase/collections-models';
	import type { PocketbaseQueryExpandOption, PocketbaseQueryResponse } from '@/pocketbase/query';
	import type { GenericRecord } from '@/utils/types';

	export type CollectionFieldModeProp = { mode?: 'search' | 'select' };

	export type CollectionFieldOptions<
		C extends CollectionName,
		Expand extends PocketbaseQueryExpandOption<C>
	> = {
		multiple?: boolean;
	} & CollectionFieldModeProp &
		Omit<CollectionInputProps<C, Expand>, 'collection'> &
		Partial<FieldOptions>;
</script>

<script
	lang="ts"
	generics="Data extends GenericRecord, C extends CollectionName, E extends PocketbaseQueryExpandOption<C> = never"
>
	import type { Writable } from 'svelte/store';
	import type { FormPath, SuperForm } from 'sveltekit-superforms';

	import { fieldProxy } from 'sveltekit-superforms/client';

	import type { FieldOptions } from '@/forms/fields/types';

	import ArrayOrItemManager from '@/components/ui-custom/arrayOrItemManager.svelte';
	import List from '@/components/ui-custom/list.svelte';
	import ListItem from '@/components/ui-custom/listItem.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Form from '@/components/ui/form';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { ensureArray, maybeArrayIsValue } from '@/utils/other';

	import type { CollectionInputProps } from './types';

	import { CollectionSelect } from '.';
	import CollectionSearch from './collectionSearch.svelte';
	import { createDefaultRecordPresenter, createRecordDisplay } from './utils';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPath<Data>;
		collection: C;
		options?: CollectionFieldOptions<C, E>;
	}

	const { form, name, collection, options = {} }: Props = $props();

	const { mode = 'select', displayFn, multiple = false } = $derived(options);

	const defaultPresenter = $derived(createDefaultRecordPresenter(collection));
	const presenter = $derived(displayFn ?? defaultPresenter);

	const queryOptions = $derived(options.queryOptions ?? {});
	const exclude = $derived(queryOptions.exclude ?? []);

	//

	const Component = $derived(mode == 'search' ? CollectionSearch : CollectionSelect);

	const valueProxy = $derived(fieldProxy(form, name)) as Writable<string | string[] | undefined>;

	function fetchRecord(collection: C, id: string): Promise<PocketbaseQueryResponse<C, E>> {
		return pb.collection(collection).getOne(id, { requestKey: null });
	}

	function addItem(proxy: typeof valueProxy, value: string, multiple: boolean) {
		proxy.update((v) => {
			if (multiple) {
				if (Array.isArray(v)) {
					if (v.includes(value)) return v;
					else return [...v, value];
				} else return [value];
			} else {
				return value;
			}
		});
	}
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children({ props })}
			<Component
				{collection}
				queryOptions={{
					...queryOptions,
					exclude: [...exclude, ...ensureArray($valueProxy)]
				}}
				displayFn={presenter}
				displayFields={options.displayFields}
				onSelect={(record) => {
					addItem(valueProxy, record.id, multiple);
				}}
				clearValueOnSelect
				controlAttrs={props}
			/>

			<List class="min-h-[42px]">
				{#if maybeArrayIsValue($valueProxy)}
					<ArrayOrItemManager bind:data={$valueProxy}>
						{#snippet children({ item, removeItem })}
							<ListItem onclick={removeItem}>
								{#await fetchRecord(collection, item) then record}
									{createRecordDisplay(record, options.displayFields, presenter)}
								{/await}
							</ListItem>
						{/snippet}
					</ArrayOrItemManager>
				{:else}
					<ListItem hideButton class="h-10 !justify-center">
						<T tag="small" class="text-secondary-foreground/30 font-normal">
							{m.No_items_selected()}
						</T>
					</ListItem>
				{/if}
			</List>
		{/snippet}
	</FieldWrapper>
</Form.Field>
