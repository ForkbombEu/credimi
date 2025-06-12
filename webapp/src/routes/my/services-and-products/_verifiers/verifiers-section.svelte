<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import { m } from '@/i18n';
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import VerifierStandardVersionField from './verifier-standard-version-field.svelte';
	import { CheckboxField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import VerifierCard from './verifier-card.svelte';

	//

	type Props = {
		organizationId?: string;
	};

	let { organizationId }: Props = $props();
</script>

<CollectionManager
	collection="verifiers"
	queryOptions={{
		filter: `owner.id = '${organizationId}'`,
		expand: ['credentials']
	}}
	formFieldsOptions={{
		exclude: ['owner', 'conformance_checks'],
		snippets: {
			standard_and_version,
			published,
			description
		},
		labels: {
			credentials: m.Linked_credentials()
		},
		relations: {
			credentials: {
				displayFields: ['issuer_name', 'name', 'key', 'format']
			}
		},
		order: ['published']
	}}
>
	{#snippet top({ Header })}
		<Header title="Verifiers"></Header>
	{/snippet}

	{#snippet records({ records })}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each records as verifier}
				<VerifierCard {verifier} credentials={verifier.expand?.credentials} />
			{/each}
		</div>
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
