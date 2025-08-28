<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type {
		FieldSnippetOptions,
		FieldsOptions
	} from '@/collections-components/form/collectionFormTypes';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';

	//

	export function options(
		organizationId: string,
		verifierId: string
	): Partial<FieldsOptions<'use_cases_verifications'>> {
		return {
			hide: {
				owner: organizationId,
				verifier: verifierId
			},
			descriptions: {
				name: m.verifier_field_description_cryptographic_binding_methods(),
				description: m.use_case_verification_field_description_description(),
				deeplink: m.use_case_verification_field_description_deeplink(),
				credentials: m.use_case_verification_field_description_credentials(),
				published: m.use_case_verification_field_description_published()
			},
			order: ['name', 'deeplink', 'credentials', 'description', 'published'],
			relations: {
				credentials: {
					mode: 'select',
					displayFields: ['issuer_name', 'name', 'key']
				}
			},
			snippets: {
				description
			}
		};
	}
</script>

{#snippet description({ form }: FieldSnippetOptions<'use_cases_verifications'>)}
	<MarkdownField {form} name="description" />
{/snippet}
