<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { BlocksIcon } from '@lucide/svelte';
	import { Pipeline } from '$lib';
	import SectionCard from '$lib/layout/section-card.svelte';
	import z from 'zod/v3';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';
	import type { PipelinesRecord } from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import CodeEditorField from '@/forms/fields/codeEditorField.svelte';
	import { goto, m } from '@/i18n';

	import PipelineFormLayout from './pipeline-form-layout.svelte';

	//

	type Props = {
		recordId?: string;
		initialData?: Partial<PipelinesRecord>;
	};

	let { recordId, initialData }: Props = $props();

	const title = $derived.by(() => {
		if (recordId) {
			return m.Edit_pipeline();
		} else {
			return m.New_pipeline();
		}
	});

	//

	function refineAsPipelineYaml(schema: z.ZodString | z.ZodOptional<z.ZodString>) {
		return schema.superRefine((v, ctx) => {
			if (!v) return;
			const result = Pipeline.validateYaml(v);
			if (!result.ok) {
				ctx.addIssue({ code: z.ZodIssueCode.custom, message: result.message });
			}
		});
	}
</script>

<PipelineFormLayout {title}>
	{#snippet topbarRight()}
		{#if !recordId}
			<Button href="/my/pipelines/new" variant="ghost">
				<BlocksIcon />
				{m.blocks_mode()}
			</Button>
		{/if}
	{/snippet}
	<SectionCard class="mx-auto w-full max-w-7xl">
		<CollectionForm
			collection="pipelines"
			{recordId}
			{initialData}
			fieldsOptions={{
				exclude: ['owner', 'canonified_name', 'published', 'manual'],
				hide: { steps: '[]' },
				snippets: { yaml }
			}}
			beforeSubmit={(data) => ({ ...data, manual: true })}
			onSuccess={async () => {
				await goto('/my/pipelines');
			}}
			refineSchema={(schema) =>
				schema.extend({
					yaml: refineAsPipelineYaml(z.string()) as unknown as z.ZodString
				})}
		/>
	</SectionCard>
</PipelineFormLayout>

{#snippet yaml({ form }: FieldSnippetOptions<'pipelines'>)}
	<CodeEditorField {form} name="yaml" options={{ lang: 'yaml', minHeight: 600 }} />
{/snippet}
