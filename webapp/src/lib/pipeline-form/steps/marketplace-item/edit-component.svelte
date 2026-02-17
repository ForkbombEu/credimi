<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { MarketplaceItem } from '$lib/marketplace/types.js';

	import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
	import * as steps from '$lib/pipeline-form/steps';
	import { getPath } from '$lib/utils/index.js';
	import { resource } from 'runed';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes.js';
	import type {
		CollectionRecords,
		CredentialsResponse,
		CustomChecksResponse,
		UseCasesVerificationsResponse
	} from '@/pocketbase/types/index.generated.js';

	import { CollectionForm } from '@/collections-components/index.js';
	import { CodeEditorField } from '@/forms/fields/index.js';

	//

	let { data, closeDialog }: steps.EditComponentProps<MarketplaceItem> = $props();

	//

	type Item = CredentialsResponse | CustomChecksResponse | UseCasesVerificationsResponse;
	type CollectionName = 'credentials' | 'custom_checks' | 'use_cases_verifications';
	type FieldProps = FieldSnippetOptions<CollectionName>;

	const record = resource(
		() => getPath(data),
		async () => {
			const res = await getRecordByCanonifiedPath<Item>(getPath(data));
			console.log(res);
			if (res instanceof Error) {
				console.error(res);
				return;
			}
			console.log(res.collectionName);
			return res;
		}
	);

	//

	type ExcludedFields = {
		[C in CollectionName]: (keyof CollectionRecords[C])[];
	};

	const excludedFieldsConfig: ExcludedFields = {
		credentials: ['canonified_name', 'conformant', 'published', 'credential_issuer'],
		custom_checks: ['canonified_name', 'owner', 'published'],
		use_cases_verifications: ['canonified_name', 'owner', 'published']
	};

	const excludedFields = $derived.by(() => {
		if (!record.current) return [];
		return excludedFieldsConfig[record.current.collectionName as CollectionName] as never[];
	});
</script>

{#if record.current}
	<CollectionForm
		collection={record.current.collectionName as CollectionName}
		recordId={record.current.id}
		initialData={record.current}
		fieldsOptions={{
			exclude: excludedFields,
			snippets: { yaml }
		}}
		onSuccess={() => {
			closeDialog();
		}}
	/>
{/if}

{#snippet yaml({ form }: FieldProps)}
	<CodeEditorField {form} name="yaml" options={{ lang: 'yaml' }} />
{/snippet}
