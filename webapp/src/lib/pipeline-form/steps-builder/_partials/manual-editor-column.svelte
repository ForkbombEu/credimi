<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';

	import type { InlineManualEditor } from '../inline-manual-editor.svelte.js';

	type Props = {
		editor: InlineManualEditor;
	};

	let { editor }: Props = $props();
</script>

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
			class="sticky bottom-0 z-10 border-t border-destructive/30 bg-white px-4 py-2 text-sm text-destructive shadow-[0_-4px_12px_rgba(0,0,0,0.08)]"
		>
			{editor.validation.message}
		</div>
	{/if}
</div>
