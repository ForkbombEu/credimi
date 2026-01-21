<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T">
	import { ArrowDown, ArrowUp } from '@lucide/svelte';
	import { capitalize } from 'lodash';

	import type { KeyOf } from '@/utils/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { Head } from '@/components/ui/table';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	interface Props {
		field: KeyOf<T>;
		label?: string | undefined;
	}

	let { field, label = undefined }: Props = $props();

	const { manager } = getCollectionManagerContext();

	const isSortField = $derived(manager.query.hasSort(field));
	const sort = $derived(manager.query.getSort(field));

	async function handleClick() {
		if (!isSortField) {
			manager.query.setSort(field, 'ASC');
		} else if (sort) {
			manager.query.flipSort(sort);
		}
	}
</script>

<Head class="group px-4!">
	<div class="flex items-center gap-x-2">
		{label ?? capitalize(field)}
		<Button
			size="icon"
			variant="ghost"
			class={[isSortField ? 'visible' : 'invisible', 'size-6 group-hover:visible']}
			onclick={handleClick}
		>
			<Icon
				src={!isSortField ? ArrowUp : sort?.[1] == 'DESC' ? ArrowDown : ArrowUp}
				size={14}
			/>
		</Button>
	</div>
</Head>
