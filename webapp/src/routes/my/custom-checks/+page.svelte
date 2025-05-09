<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionManager } from '@/collections-components';
	import { CodeEditorField, SelectField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { currentUser, pb } from '@/pocketbase';
	import { yaml } from '@codemirror/lang-yaml';
	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import T from '@/components/ui-custom/t.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';

	//

	let { data } = $props();

	const options = $derived(
		data.standardsAndTestSuites.flatMap((standard) =>
			standard.versions.map((version) => ({
				value: `${standard.uid}/${version.uid}`,
				label: `${standard.name} – ${version.name}`
			}))
		)
	);
</script>

<div class="space-y-4">
	<CollectionManager
		collection="custom_checks"
		formFieldsOptions={{
			// TODO - Enforce owner from backend
			hide: {
				owner: $currentUser?.id
			},
			exclude: ['organization'],
			snippets: {
				yaml: yamlField,
				standard_and_version: standardAndVersionField
			}
		}}
	>
		{#snippet top({ Header })}
			<Header title={m.Custom_checks()}></Header>
		{/snippet}

		{#snippet records({ records, Card })}
			<div class="space-y-2">
				{#each records as record}
					{@const logo = pb.files.getURL(record, record.logo)}
					<Card {record} class="bg-background !pl-4" hide={['share', 'select']}>
						<div class="flex items-start gap-4">
							<Avatar
								src={logo}
								class="rounded-sm"
								fallback={record.name.slice(0, 2)}
							/>
							<div>
								<T class="font-bold">{record.name}</T>
								<T class="mb-2 font-mono text-xs">{record.standard_and_version}</T>
								<T class="text-sm text-gray-400">{record.description}</T>
							</div>
						</div>
					</Card>
				{/each}
			</div>
		{/snippet}
	</CollectionManager>
</div>

<!--  -->

{#snippet yamlField({ form }: FieldSnippetOptions<'custom_checks'>)}
	<CodeEditorField {form} name="yaml" options={{ lang: yaml() }} />
{/snippet}

{#snippet standardAndVersionField({ form }: FieldSnippetOptions<'custom_checks'>)}
	<SelectField
		{form}
		name="standard_and_version"
		options={{
			items: [...options]
		}}
	/>
{/snippet}
