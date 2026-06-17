<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { BlocksIcon } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import { m } from '@/i18n';

	import type { InlineManualEditor } from '../inline-manual-editor.svelte.js';
	import type { StepsBuilder } from '../steps-builder.svelte.js';

	import Column from './column.svelte';

	type Props = {
		builder: StepsBuilder;
		editor: InlineManualEditor;
	};

	let { builder, editor }: Props = $props();
</script>

<Column title={m.manual_edit()} class="card min-w-0 basis-2 overflow-hidden">
	{#snippet titleRight()}
		<Button variant="outline" size="sm" onclick={() => void builder.exitManualMode()}>
			<BlocksIcon />
			{m.back_to_steps()}
		</Button>
	{/snippet}

	<div class="relative flex min-h-0 min-w-0 grow flex-col">
		<CodeEditor
			lang="yaml"
			bind:value={editor.yaml}
			minHeight={null}
			class="min-h-0 grow rounded-none"
			hideCopyButton
		/>
		{#if !editor.validation.ok}
			<div
				class="sticky bottom-0 border-t bg-destructive/10 px-4 py-2 text-sm text-destructive"
			>
				{editor.validation.message}
			</div>
		{/if}
	</div>
</Column>
