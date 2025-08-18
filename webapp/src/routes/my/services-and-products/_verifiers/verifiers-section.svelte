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
		organizationId: string;
		id?: string;
	};

	let { organizationId, id }: Props = $props();
</script>

<CollectionManager
	collection="verifiers"
	queryOptions={{
		filter: `owner.id = '${organizationId}'`,
		expand: ['use_cases_verifications_via_verifier'],
		sort: ['created', 'DESC']
	}}
	formFieldsOptions={{
		exclude: ['owner', 'conformance_checks', 'published'],
		snippets: {
			standard_and_version,
			description
		},
		descriptions: {
			name: m.verifier_field_description_name(),
			description: m.verifier_field_description_description(),
			logo: m.verifier_field_description_logo(),
			url: m.verifier_field_description_url(),
			repository_url: m.verifier_field_description_repository_url(),
			standard_and_version: m.verifier_field_description_standard_and_version(),
			format: m.verifier_field_description_format(),
			signing_algorithms: m.verifier_field_description_signing_algorithms(),
			cryptographic_binding_methods:
				m.verifier_field_description_cryptographic_binding_methods()
		},
		order: []
	}}
>
	{#snippet top({ Header })}
		<Header title="Verifiers" {id}>
			{#snippet buttonContent()}
				{m.Create_verifier()}
			{/snippet}
		</Header>
	{/snippet}

	{#snippet records({ records })}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each records as verifier}
				{@const useCasesVerifications =
					verifier.expand?.use_cases_verifications_via_verifier ?? []}
				<VerifierCard {verifier} {useCasesVerifications} {organizationId} />
			{/each}
		</div>
	{/snippet}
</CollectionManager>

{#snippet standard_and_version({ form }: FieldSnippetOptions<'verifiers'>)}
	<VerifierStandardVersionField {form} />
{/snippet}

{#snippet description({ form }: FieldSnippetOptions<'verifiers'>)}
	<MarkdownField {form} name="description" />
{/snippet}
