<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type {
		FieldSnippetOptions,
		FieldsOptions
	} from '@/collections-components/form/collectionFormTypes';

	import QrGenerationField from '@/components/qr-generation-field.svelte';
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
				yaml: m.YAML_Configuration_section_description(),
				credentials: m.use_case_verification_field_description_credentials(),
				published: m.use_case_verification_field_description_published()
			},
			order: ['name', 'yaml', 'credentials', 'description'],
			relations: {
				credentials: {
					mode: 'select',
					displayFields: ['name']
				}
			},
			exclude: ['published', 'canonified_name'],
			snippets: {
				description,
				yaml: yaml_editor
			}
		};
	}
</script>

{#snippet description({ form }: FieldSnippetOptions<'use_cases_verifications'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet yaml_editor({ form }: FieldSnippetOptions<'use_cases_verifications'>)}
	<div>
		<QrGenerationField
			{form}
			fieldName="yaml"
			label={m.YAML_Configuration()}
			description={m.Provide_configuration_in_YAML_format()}
			placeholder={m.Run_the_code_to_generate_QR_code()}
			successMessage={m.Test_Completed_Successfully()}
			loadingMessage={m.Running_test()}
			enableStructuredErrors={true}
		/>
	</div>
{/snippet}
