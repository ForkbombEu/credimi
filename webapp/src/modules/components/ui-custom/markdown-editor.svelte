<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Carta, MarkdownEditor } from 'carta-md';
	import DOMPurify from 'dompurify';
	import 'carta-md/default.css';
	import { onMount } from 'svelte';

	//

	type Props = {
		value?: string;
	};

	let { value = $bindable() }: Props = $props();

	//

	const carta = new Carta({
		sanitizer: DOMPurify.sanitize,
		disableIcons: ['taskList']
	});

	// Fixes style not being applied to the rendered markdown
	onMount(() => {
		document.querySelectorAll('.carta-renderer.markdown-body > div').forEach((div) => {
			div.classList.add('prose');
		});
	});
</script>

<MarkdownEditor bind:value {carta} />

<style lang="postcss">
	:global(.carta-input),
	:global(.carta-renderer) {
		min-height: 120px;
		max-height: 400px;
		overflow: auto;
	}
</style>
