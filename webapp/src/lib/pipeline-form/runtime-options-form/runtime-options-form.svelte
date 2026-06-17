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
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import CodeEditorField from '@/forms/fields/codeEditorField.svelte';
	import Form from '@/forms/form.svelte';
	import { m } from '@/i18n/index.js';

	import type { RuntimeOptionsForm } from './runtime-options-form.svelte.js';

	//

	const { self: form }: SelfProp<RuntimeOptionsForm> = $props();
</script>

<Dialog bind:open={form.isOpen}>
	{#snippet trigger({ props: dialogProps })}
		<Tooltip disabled={!form.disabled}>
			{#snippet child({ props: tooltipProps })}
				<span {...tooltipProps} class="inline-flex">
					<Button variant="outline" {...dialogProps} disabled={form.disabled}>
						<HourglassIcon />
						{m.parameters()}
					</Button>
				</span>
			{/snippet}
			{#snippet content()}{m.unavailable_in_manual_edit()}{/snippet}
		</Tooltip>
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
