<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { HourglassIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import CodeEditorField from '@/forms/fields/codeEditorField.svelte';
	import Form from '@/forms/form.svelte';
	import { m } from '@/i18n/index.js';

	import type { ActivityOptionsForm } from './activity-options-form.svelte.js';

	//

	const { self: form }: SelfProp<ActivityOptionsForm> = $props();
</script>

<Dialog bind:open={form.isOpen}>
	{#snippet trigger({ props })}
		<Button variant="outline" {...props}>
			<HourglassIcon />
			{m.parameters()}
		</Button>
	{/snippet}

	{#snippet content()}
		{@const f = form.mountForm()}
		<div class="space-y-6">
			<T tag="h4">{m.parameters()}</T>

			<Form form={f}>
				<CodeEditorField
					form={f}
					name="code"
					options={{
						label: m.YAML_Configuration(),
						lang: 'yaml',
						minHeight: 200
					}}
				/>
			</Form>
		</div>
	{/snippet}
</Dialog>
