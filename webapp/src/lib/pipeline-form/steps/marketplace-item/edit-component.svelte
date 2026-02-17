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
			if (res instanceof Error) {
				console.error(res);
				return;
			}
			return res;
		}
	);
</script>

{#if record.current}
	<CollectionForm
		collection={record.current.collectionName as CollectionName}
		recordId={record.current.id}
		initialData={record.current}
		fieldsOptions={{
			include: ['yaml'],
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
