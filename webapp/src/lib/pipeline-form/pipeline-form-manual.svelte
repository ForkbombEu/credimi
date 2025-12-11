<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import SectionCard from '$lib/layout/section-card.svelte';
	import { BlocksIcon } from 'lucide-svelte';

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

	// const ajv = new Ajv({ allowUnionTypes: true, dynamicRef: true });
	// const validatePipeline = ajv.compile(PipelineSchema);

	// export function refineAsPipelineYaml(schema: z.ZodString | z.ZodOptional<z.ZodString>) {
	// 	return schema.superRefine((v, ctx) => {
	// 		if (!v) return;

	// 		let res: unknown;
	// 		try {
	// 			res = parseYaml(v);
	// 		} catch (e) {
	// 			ctx.addIssue({
	// 				code: z.ZodIssueCode.custom,
	// 				message: `Invalid YAML document: ${getExceptionMessage(e)}`
	// 			});
	// 			return;
	// 		}

	// 		const isValid = validatePipeline(res);
	// 		if (!isValid) {
	// 			const error = ajv.errorsText(validatePipeline.errors);
	// 			ctx.addIssue({
	// 				code: z.ZodIssueCode.custom,
	// 				message: `Invalid YAML document: ${error}`
	// 			});
	// 		}
	// 	});
	// }
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
				exclude: ['owner', 'canonified_name', 'published'],
				hide: { steps: '[]' },
				snippets: { yaml }
			}}
			onSuccess={async () => {
				await goto('/my/pipelines');
			}}
		/>
	</SectionCard>
</PipelineFormLayout>

{#snippet yaml({ form }: FieldSnippetOptions<'pipelines'>)}
	<CodeEditorField {form} name="yaml" options={{ lang: 'yaml', minHeight: 600 }} />
{/snippet}
