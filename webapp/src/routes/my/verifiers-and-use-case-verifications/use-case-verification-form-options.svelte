<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import CollectionLogoField from '$lib/components/collection-logo-field.svelte';
	import QrFieldWrapper from '$lib/layout/qr-field-wrapper.svelte';
	import { stepciYamlSchema } from '$lib/utils';
	import { z } from 'zod/v3';

	import type {
		CollectionFormProps,
		FieldSnippetOptions
	} from '@/collections-components/form/collectionFormTypes';

	import QrGenerationField from '@/components/qr-generation-field.svelte';
	import CodeEditorField from '@/forms/fields/codeEditorField.svelte';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';

	//

	export function options(
		organizationId: string,
		verifierId: string
	): Partial<CollectionFormProps<'use_cases_verifications'>> {
		return {
			refineSchema: (schema) =>
				schema.extend({
					yaml: stepciYamlSchema as unknown as z.ZodString
				}),
			fieldsOptions: {
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
				order: ['name', 'description', 'yaml', 'credentials'],
				relations: {
					credentials: {
						mode: 'select',
						displayFields: ['name']
					}
				},
				exclude: ['published', 'canonified_name'],
				snippets: {
					description,
					yaml: yaml_editor,
					logo,
					dcql_query
				}
			}
		};
	}
</script>

{#snippet description({ form }: FieldSnippetOptions<'use_cases_verifications'>)}
	<MarkdownField {form} name="description" />
{/snippet}

{#snippet yaml_editor({ form }: FieldSnippetOptions<'use_cases_verifications'>)}
	<QrFieldWrapper label={m.YAML_Configuration()} required class="!p-3">
		<QrGenerationField
			{form}
			fieldName="yaml"
			description={m.Provide_configuration_in_YAML_format()}
			placeholder={m.Run_the_code_to_generate_QR_code()}
			successMessage={m.Test_Completed_Successfully()}
			loadingMessage={m.Running_test()}
			enableStructuredErrors={true}
			hideLabel
		/>
	</QrFieldWrapper>
{/snippet}

{#snippet logo()}
	<CollectionLogoField />
{/snippet}

{#snippet dcql_query({ form }: FieldSnippetOptions<'use_cases_verifications'>)}
	<CodeEditorField
		{form}
		name="dcql_query"
		options={{
			lang: 'json',
			minHeight: 300,
			label: m.DCQL_Query()
		}}
	/>
{/snippet}
