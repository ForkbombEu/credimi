<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script
	lang="ts"
	generics="C extends CollectionName, E extends PocketbaseQueryExpandOption<C> = never"
>
	import type { SearchFunction } from '@/components/ui-custom/search.svelte';
	import type { CollectionName } from '@/pocketbase/collections-models';

	import Search from '@/components/ui-custom/search.svelte';
	import {
		type PocketbaseQueryResponse,
		type PocketbaseQueryExpandOption,
		type PocketbaseQueryOptions,
		PocketbaseQueryAgent
	} from '@/pocketbase/query';

	import type { CollectionInputProps } from './types';

	import { createRecordDisplay } from './utils';

	//

	type Props = CollectionInputProps<C, E>;

	let {
		collection,
		queryOptions = {},
		disabled = false,
		label = undefined,
		placeholder = undefined,
		onSelect = () => {},
		displayFields = undefined,
		displayFn = undefined,
		...rest
	}: Props = $props();

	//

	type SearchFn = SearchFunction<PocketbaseQueryResponse<C, E>>;

	const searchFunction: SearchFn = $derived(async function (text: string | undefined) {
		const query: PocketbaseQueryOptions<C, E> = { ...queryOptions, search: text };

		const runners = new PocketbaseQueryAgent({ collection, ...query }, { requestKey: null });
		const records = await runners.getFullList();

		return records.map((item) => ({
			value: item,
			label: createRecordDisplay(item, displayFields, displayFn),
			disabled: false
		}));
	});
</script>

<Search searchFn={searchFunction} {onSelect} {label} {placeholder} {disabled} {...rest} />
