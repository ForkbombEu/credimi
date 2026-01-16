<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { PencilIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Field, SwitchField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import Form from '@/forms/form.svelte';
	import { m } from '@/i18n';

	import type { MetadataForm } from './metadata-form.svelte.js';

	//

	const { self: form }: SelfProp<MetadataForm> = $props();
</script>

<Dialog bind:open={form.isOpen} title={m.Metadata()}>
	{#snippet trigger({ props })}
		<Button variant="outline" {...props}>
			<PencilIcon />
			{m.Info()}
		</Button>
	{/snippet}

	{#snippet content()}
		{@const f = form.mountForm()}
		<div>
			<T class="text-muted-foreground mb-6">{m.save_pipeline_description()}</T>
			<Form form={f}>
				<div class="flex items-start gap-6">
					<div class="grow">
						<Field form={f} name="name" options={{ label: m.Name() }} />
					</div>
					<div class="pt-10">
						<SwitchField
							form={f}
							name="published"
							options={{ label: m.Publish_to_marketplace() }}
						/>
					</div>
				</div>
				<MarkdownField form={f} name="description" options={{ label: m.Description() }} />
			</Form>
		</div>
	{/snippet}
</Dialog>
