<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ClipboardCopy, Check } from 'lucide-svelte';
	import Button from '@/components/ui/button/button.svelte';
	import { onMount } from 'svelte';
	import { codeToHtml } from 'shiki';
	import { stat } from 'fs';

	type Props = {
		content: string;
		class?: string;
		showCopyButton?: boolean;
		language?: string;
		containerClass?: string;
	};

	let {
		content,
		class: className = '',
		showCopyButton = true,
		language = '',
		containerClass = ''
	}: Props = $props();

	let isCopied = $state(false);
	let highlighted = $state('');

	// Generate highlighted HTML on mount
	onMount(async () => {
		if (content) {
			highlighted = await codeToHtml(content, {
				lang: language,
				theme: 'vitesse-dark'
			});
		}
	});

	async function copyToClipboard() {
		if (!content) return;

		try {
			await navigator.clipboard.writeText(content);
			isCopied = true;
			setTimeout(() => {
				isCopied = false;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy text: ', err);
		}
	}

	// Use provided classes or sensible defaults
	const preClasses = $derived(
		className || 'rounded-lg border border-slate-200 bg-white p-4 overflow-x-auto text-sm'
	);
</script>

<div class="relative {containerClass}">
	<pre
		class="{preClasses} relative"
		class:language-json={language === 'json'}
		class:language-yaml={language === 'yaml'}>
		{highlighted || content}
		{#if showCopyButton && content}
			<Button
				type="button"
				variant="ghost"
				size="sm"
				class="absolute right-2 top-2 z-10 h-6 w-6 border border-slate-300/50 bg-white/90 p-0 opacity-80 shadow-sm backdrop-blur-sm hover:bg-white/100 hover:opacity-100"
				onclick={copyToClipboard}
				title={isCopied ? 'Copied!' : 'Copy to clipboard'}>
				{#if isCopied}
					<Check class="h-3 w-3 text-green-600" />
				{:else}
					<ClipboardCopy class="h-3 w-3 text-slate-600" size={16} />
				{/if}
			</Button>
		{/if}
	</pre>
</div>
