<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ConfigField } from '$start-checks-form/types';

	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { Eye, Pencil, Undo } from 'lucide-svelte';

	import Label from '@/components/ui/label/label.svelte';
	import * as Popover from '@/components/ui/popover';
	import { Form } from '@/forms';
	import { m } from '@/i18n';

	import type { DependentCheckConfigFormEditor } from './dependent-check-config-form-editor.svelte.js';

	import ConfigFormInput from './check-config-form-input.svelte';

	//

	type Props = {
		form: DependentCheckConfigFormEditor;
	};

	let { form }: Props = $props();

	function previewValue(value: unknown, type: ConfigField['field_type']): string {
		const NULL_VALUE = '<null>';
		if (!value) return NULL_VALUE;
		if (type === 'string') return value as string;
		if (type === 'object') return JSON.stringify(JSON.parse(value as string), null, 4);
		return NULL_VALUE;
	}
</script>

<Form form={form.superform} hide={['submit_button']} hideRequiredIndicator>
	{#each form.independentFields as field}
		<ConfigFormInput {field} form={form.superform} />
	{/each}

	{#each form.overriddenFields as field}
		<ConfigFormInput {field} form={form.superform}>
			{#snippet labelRight()}
				<button
					class="text-primary flex items-center gap-2 text-sm underline hover:no-underline"
					onclick={(e) => {
						e.preventDefault(); // Important to prevent form submission
						form.resetOverride(field.credimi_id);
					}}
				>
					<Undo size={14} />
					{m.Reset_to_default()}
				</button>
			{/snippet}
		</ConfigFormInput>
	{/each}

	{#if form.dependentFields.length}
		<div class="space-y-2">
			<Label>{m.Default_fields()}</Label>
			<ul class="space-y-1">
				{#each form.dependentFields as { credimi_id, field_label, field_type }}
					{@const value = form.props.formDependency.getData()[credimi_id]}
					{@const valuePreview = previewValue(value, field_type)}

					<li class="flex items-center gap-2">
						<span class="font-mono text-sm">{field_label}</span>

						<Popover.Root>
							<Popover.Trigger
								class="rounded-md p-1 hover:cursor-pointer hover:bg-gray-200"
							>
								<Eye size={14} />
							</Popover.Trigger>
							<Popover.Content class="dark overflow-auto">
								{#if field_type === 'object'}
									<CodeDisplay
										content={valuePreview}
										language="json"
										class="text-xs"
									/>
								{:else}
									<pre class="text-xs">{valuePreview}</pre>
								{/if}
							</Popover.Content>
						</Popover.Root>

						<button
							class="rounded-md p-1 hover:cursor-pointer hover:bg-gray-200"
							onclick={(e) => {
								e.preventDefault(); // Important to prevent form submission
								form.overrideField(credimi_id);
							}}
						>
							<Pencil size={14} />
						</button>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</Form>
