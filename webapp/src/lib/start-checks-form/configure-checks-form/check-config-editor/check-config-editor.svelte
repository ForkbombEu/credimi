<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { CheckConfigEditor } from './check-config-editor.svelte.js';
	import { CheckConfigCodeEditorComponent } from '../check-config-code-editor/index.js';
	import { DependentCheckConfigFormEditorComponent } from '../check-config-form-editor';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Info } from 'lucide-svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';
	import SmallSectionLabel from '$start-checks-form/_utils/small-section-label.svelte';

	//

	type Props = {
		editor: CheckConfigEditor;
	};

	const { editor }: Props = $props();
</script>

<div class="flex flex-col gap-6 md:flex-row md:gap-10">
	<div class="min-w-0 shrink-0 grow basis-1 space-y-6">
		<SmallSectionLabel>{m.Fields()}</SmallSectionLabel>

		{#if editor.mode == 'form'}
			<DependentCheckConfigFormEditorComponent form={editor.formEditor} />
		{:else}
			<div class="text-muted-foreground text-sm">
				<Alert variant="info" icon={Info}>
					{#snippet content({ Title, Description })}
						<Title class="font-bold">Info</Title>
						<Description class="mb-2">
							{m.code_configuration_is_edited_fields_are_disabled()}
						</Description>

						<Button variant="outline" onclick={() => editor.codeEditor.reset()}>
							{m.reset_code_and_use_fields()}
						</Button>
					{/snippet}
				</Alert>
			</div>
		{/if}
	</div>

	<div class="flex min-w-0 shrink-0 grow basis-1 flex-col space-y-6">
		<SmallSectionLabel>{m.YAML_Configuration()}</SmallSectionLabel>
		<CheckConfigCodeEditorComponent editor={editor.codeEditor} />
	</div>
</div>
