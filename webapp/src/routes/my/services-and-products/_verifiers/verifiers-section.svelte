<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import T from '@/components/ui-custom/t.svelte';
	import { RecordEdit, RecordDelete } from '@/collections-components/manager';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import VerifierStandardVersionField from './verifier-standard-version-field.svelte';
	import { CheckboxField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';

	//

	type Props = {
		organizationId?: string;
	};

	let { organizationId }: Props = $props();
</script>

<CollectionManager
	collection="verifiers"
	queryOptions={{
		filter: `owner.id = '${organizationId}'`
	}}
	formFieldsOptions={{
		exclude: ['owner', 'conformance_checks'],
		snippets: {
			standard_and_version,
			published,
			description
		},
		order: ['published']
	}}
>
	{#snippet top({ Header })}
		<Header title="Verifiers"></Header>
	{/snippet}

	{#snippet records({ records, Card })}
		{#each records as record}
			<Card {record} class="bg-background" hide="all">
				<div class="flex items-center justify-between">
					<div>
						<T class="font-bold">{record.name}</T>
						<T>{record.url}</T>
					</div>
					<div>
						<RecordEdit {record} />
						<RecordDelete {record} />
					</div>
				</div>
				<Separator />
				<div>
					<T>{m.Linked_credentials()}</T>
					{#if record.credentials.length === 0}
						<T class="text-gray-300">{m.No_credentials_available()}</T>
					{:else}
						<T>{record.credentials.length}</T>
					{/if}
				</div>
			</Card>
		{/each}
	{/snippet}
</CollectionManager>

{#snippet standard_and_version({ form }: FieldSnippetOptions<'verifiers'>)}
	<VerifierStandardVersionField {form} />
{/snippet}

{#snippet published({ form }: FieldSnippetOptions<'verifiers'>)}
	<div class="flex justify-end gap-2">
		<CheckboxField {form} name="published" options={{ label: m.Published() }} />
	</div>
{/snippet}

{#snippet description({ form }: FieldSnippetOptions<'verifiers'>)}
	<MarkdownField {form} name="description" />
{/snippet}
