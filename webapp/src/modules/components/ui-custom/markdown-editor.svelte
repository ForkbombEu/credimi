<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Carta, MarkdownEditor, type Plugin } from 'carta-md';
	import 'carta-md/default.css';
	import DOMPurify from 'dompurify';
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

<div style="--carta-height: {height}px;">
	<MarkdownEditor mode="tabs" bind:value {carta} />
</div>

<style lang="postcss">
	:global(.carta-input, .carta-renderer, .carta-input-wrapper) {
		height: var(--carta-height) !important;
		min-height: var(--carta-height) !important;
	}
</style>
