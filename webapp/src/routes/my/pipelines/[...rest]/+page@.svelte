<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';

	import { setupCollectionForm } from '@/collections-components/form';
	import { Form } from '@/forms';
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

<FocusPageLayout backButton={{ href: '/my/pipelines', title: m.Back() }} {title}>
	<Form {form}>
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
	</Form>
</FocusPageLayout>
