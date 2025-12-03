<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import CollectionLogoField from '$lib/components/collection-logo-field.svelte';

	import type {
		FieldSnippetOptions,
		FieldsOptions
	} from '@/collections-components/form/collectionFormTypes';

	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';

	import VerifierStandardVersionField from './verifier-standard-version-field.svelte';

	//

	export const settings: Partial<FieldsOptions<'verifiers'>> = {
		exclude: ['owner', 'conformance_checks', 'published', 'canonified_name'],
		snippets: {
			standard_and_version,
			description,
			logo
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
	};
</script>

{#snippet standard_and_version({ form }: FieldSnippetOptions<'verifiers'>)}
	<VerifierStandardVersionField {form} />
{/snippet}

{#snippet description({ form }: FieldSnippetOptions<'verifiers'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet logo()}
	<CollectionLogoField />
{/snippet}
