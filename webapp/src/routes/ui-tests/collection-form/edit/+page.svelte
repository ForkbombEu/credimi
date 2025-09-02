<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { toast } from 'svelte-sonner';

	import { CollectionForm } from '@/collections-components';

	let { data } = $props();
</script>

<CollectionForm
	collection="z_test_collection"
	recordId={data.record.id}
	initialData={data.record}
	onSuccess={(record) => {
		toast.success(JSON.stringify(record));
		invalidateAll();
	}}
></CollectionForm>
