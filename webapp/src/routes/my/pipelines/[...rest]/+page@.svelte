<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BackButton from '$lib/layout/back-button.svelte';

	import { setupCollectionForm } from '@/collections-components/form';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';

	import PipelineBuilderComponent from './logic/pipeline-builder.svelte';
	import { PipelineBuilder } from './logic/pipeline-builder.svelte.js';

	//

	let { data } = $props();
	const { mode, record } = $derived(data);

	//

	const title = $derived(mode === 'create' ? m.New_pipeline() : m.Edit_pipeline());

	const form = setupCollectionForm({
		collection: 'pipelines',
		recordId: record?.id,
		initialData: record,
		fieldsOptions: {
			exclude: ['owner', 'canonified_name']
		}
	});

	const builder = new PipelineBuilder();
</script>

<div class="bg-secondary flex h-screen flex-col gap-4 p-6">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<BackButton href="/my/pipelines" class="h-6">
				{m.Back()}
			</BackButton>
			<Separator orientation="vertical" class="self-stretch bg-slate-400" />
			<T tag="h3">{title}</T>
		</div>

		<Button>{m.Save()}</Button>
	</div>
	<!-- <PageCardSection title={m.Metadata()}>
			<div class="flex items-start gap-6">
				<div class="grow">
					<Field {form} name="name" options={{ label: m.Name() }} />
				</div>
				<div class="pt-10">
					<SwitchField
						{form}
						name="published"
						options={{ label: m.Publish_to_marketplace() }}
					/>
				</div>
			</div>
			<MarkdownField {form} name="description" options={{ label: m.Description() }} />
		</PageCardSection> -->

	<PipelineBuilderComponent {builder} />
</div>
