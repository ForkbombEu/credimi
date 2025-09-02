<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import type { CollectionName } from '@/pocketbase/collections-models';
	import type { CollectionResponses } from '@/pocketbase/types';

	import { Checkbox } from '@/components/ui/checkbox';

	import { getCollectionManagerContext } from '../collectionManagerContext';

	//

	interface Props {
		record: CollectionResponses[C];
	}

	const { record }: Props = $props();

	//

	const { manager } = $derived(getCollectionManagerContext());
	const checked = $derived(manager.selectedRecords.includes(record.id));

	function handleChecked(checked: boolean | 'indeterminate') {
		if (checked == 'indeterminate') return;
		if (checked) manager.selectRecord(record.id);
		else manager.deselectRecord(record.id);
	}
</script>

<Checkbox {checked} onCheckedChange={handleChecked} value={record.id} name="select" />
