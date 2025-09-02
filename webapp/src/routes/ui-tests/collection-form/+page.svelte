<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { toast } from 'svelte-sonner';

	import { CollectionForm } from '@/collections-components';
	import FormDebug from '@/forms/components/formDebug.svelte';
</script>

<CollectionForm
	collection="z_test_collection"
	onSuccess={(record) => {
		toast.success(JSON.stringify(record));
	}}
/>

<!-- 
<div class="mx-auto max-w-xl p-4">
	<RecordForm
		{recordType}
		collection={Collections.CrudExample}
		fieldsSettings={{
			relations: {
				relation: {
					displayFields: ['name', 'active'],
					inputMode: 'select'
				},
				relation_single: { displayFields: ['name'], inputMode: 'search' },
				owner: { displayFields: ['name', 'email'], inputMode: 'search' }
			},
			hide: { owner: $currentUser?.id },
			labels: {
				text: 'Name',
				file_only_pdf_json: 'PDF or JSON file',
				boolean: 'Test check'
			},
			components: {
				textarea: createFieldComponent(Input),
				boolean: createFieldComponent(Toggle)
			}
		}}
		on:success={invalidateAll}
	/>
</div> -->
