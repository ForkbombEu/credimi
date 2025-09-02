<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Carta, MarkdownEditor, type Plugin } from 'carta-md';
	import DOMPurify from 'dompurify';
	import 'carta-md/default.css';
	import { onMount } from 'svelte';

	import MarkdownEditorIcon from './markdown-editor-image-icon.svelte';

	//

	type Props = {
		value?: string;
		height?: number;
	};

	let { value = $bindable(), height = 300 }: Props = $props();

	//

	const imagePlugin: Plugin = {
		icons: [
			{
				id: 'image',
				component: MarkdownEditorIcon,
				action: (input) => {
					const imagePlaceholder = '![alt text](image url)';
					const selection = input.getSelection();
					const isCollapsed = selection.start === selection.end;
					if (!isCollapsed) {
						input.removeAt(selection.start, selection.end - selection.start);
					}
					input.insertAt(selection.start, imagePlaceholder);
				}
			}
		]
	};

	const carta = new Carta({
		sanitizer: DOMPurify.sanitize,
		disableIcons: ['taskList'],
		extensions: [imagePlugin]
	});

	// Fixes style not being applied to the rendered markdown
	onMount(() => {
		document.querySelectorAll('.carta-renderer.markdown-body > div').forEach((div) => {
			div.classList.add('prose');
		});
	});
</script>

<div class="resizable-container" style="height: {height}px;">
	<MarkdownEditor bind:value {carta} />
</div>

<style lang="postcss">
	.resizable-container {
		resize: both;
		overflow: auto;
		min-height: 120px;
		min-width: 200px;
		border: 1px solid hsl(var(--border));
		border-radius: 6px;
		width: 100%;
	}

	:global(.carta-input),
	:global(.carta-renderer) {
		min-height: 120px;
		overflow: auto;
	}
</style>
