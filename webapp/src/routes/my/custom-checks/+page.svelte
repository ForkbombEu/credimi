<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import { CodeEditorField, SelectField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';
	import type { CustomChecksFormData } from '@/pocketbase/types';
	import type { SuperForm } from 'sveltekit-superforms';
	import { yaml } from '@codemirror/lang-yaml';

	//

	const options = {
		standard_1: ['ciao', 'draft-2'],
		standard_2: ['v.0', 'v.1']
	};

	const standards = Object.keys(options).map((o) => ({ value: o, label: o }));
</script>

<CollectionManager
	collection="custom_checks"
	formFieldsOptions={{
		hide: {
			owner: $currentUser?.id
		},
		exclude: ['organization', 'standard'],
		snippets: {
			yaml: yamlField,
			standard: standardField,
			standard_version: standardVersionField
		}
	}}
>
	{#snippet top({ Header })}
		<Header title={m.Custom_checks()}></Header>
	{/snippet}
</CollectionManager>

{#snippet yamlField(data: { form: SuperForm<CustomChecksFormData>; field: string })}
	<CodeEditorField form={data.form} name="yaml" options={{ lang: yaml() }} />
{/snippet}

{#snippet standardField(data: { form: SuperForm<CustomChecksFormData>; field: string })}
	<SelectField
		form={data.form}
		name="standard"
		options={{
			items: [...standards]
		}}
	/>
{/snippet}

{#snippet standardVersionField(data: { form: SuperForm<CustomChecksFormData>; field: string })}
	<SelectField
		form={data.form}
		name="standard_version"
		options={{
			items: []
		}}
	/>
{/snippet}
