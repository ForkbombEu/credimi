<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as steps from '$lib/pipeline-form/steps';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes.js';

	import CollectionForm from '@/collections-components/form/collectionForm.svelte';
	import { CodeEditorField } from '@/forms/fields/index.js';

	import type { WalletActionStepData } from './wallet-action-step-form.svelte.js';

	//

	let { data, closeDialog }: steps.EditComponentProps<WalletActionStepData> = $props();
</script>

<CollectionForm
	collection="wallet_actions"
	recordId={data.action.id}
	initialData={data.action}
	fieldsOptions={{
		include: ['code'],
		snippets: { code }
	}}
	onSuccess={() => {
		closeDialog();
	}}
/>

{#snippet code({ form }: FieldSnippetOptions<'wallet_actions'>)}
	<CodeEditorField {form} name="code" options={{ lang: 'yaml' }} />
{/snippet}
