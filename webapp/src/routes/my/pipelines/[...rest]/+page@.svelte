<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BackButton from '$lib/layout/back-button.svelte';
	import { jsonStringSchema } from '$lib/utils';
	import { RedoIcon, SaveIcon, UndoIcon } from 'lucide-svelte';
	import { nanoid } from 'nanoid';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';

	import Dialog from '@/components/ui-custom/dialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Separator } from '@/components/ui/separator';
	import { createForm, Form } from '@/forms';
	import { Field, SwitchField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { goto, m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import PipelineBuilderComponent from './logic/pipeline-builder.svelte';
	import { PipelineBuilder } from './logic/pipeline-builder.svelte.js';

	//

	let { data } = $props();
	const { mode, record } = $derived(data);

	//

	const title = $derived(mode === 'create' ? m.New_pipeline() : m.Edit_pipeline());

	const form = createForm({
		adapter: zod(
			z.object({
				description: z.string().min(3),
				published: z.boolean().default(false),
				name: z.string().min(3),
				steps: jsonStringSchema
			})
		),
		onSubmit: async ({ form }) => {
			try {
				await pb
					.collection('pipelines')
					.create({ ...form.data, canonified_name: nanoid(5) });
				await goto('/my/pipelines');
			} catch (error) {
				console.error(error);
			}
		}
	});

	const builder = new PipelineBuilder();

	let isFormOpen = $state(false);

	async function save() {
		const res = await form.validateForm();
		if (!res.valid) {
			isFormOpen = true;
		} else {
			form.submit();
		}
	}

	$effect(() => {
		form.form.update((data) => {
			data.steps = JSON.stringify({ steps: builder.steps });
			return data;
		});
	});
</script>

<div class="bg-secondary flex h-screen flex-col gap-4 overflow-hidden p-6">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<BackButton href="/my/pipelines" class="h-6">
				{m.Back()}
			</BackButton>
			<Separator orientation="vertical" class="self-stretch bg-slate-400" />
			<T tag="h3">{title}</T>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="outline" onclick={() => builder.undo()}>
				<UndoIcon />
				{m.Undo()}
			</Button>
			<Button variant="outline" onclick={() => builder.redo()}>
				<RedoIcon />
				{m.Redo()}
			</Button>
			<Button disabled={!builder.isReady()} onclick={save}>
				<SaveIcon />
				{m.Save()}
			</Button>
		</div>
	</div>
	<PipelineBuilderComponent {builder} />
</div>

<Dialog bind:open={isFormOpen} title={m.Save()} contentClass="w-full">
	{#snippet content()}
		<T class="text-muted-foreground mb-6">{m.save_pipeline_description()}</T>
		<Form {form}>
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
		</Form>
	{/snippet}
</Dialog>
