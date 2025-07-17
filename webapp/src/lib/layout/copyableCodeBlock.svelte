<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ClipboardCopy, Check } from 'lucide-svelte';
	import Button from '@/components/ui/button/button.svelte';

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
		class:language-yaml={language === 'yaml'}
	>{content}{#if showCopyButton && content}		<Button
			type="button"
			variant="ghost"
			size="sm"
			class="absolute top-2 right-2 h-6 w-6 p-0 opacity-80 hover:opacity-100 bg-white/90 hover:bg-white/100 backdrop-blur-sm border border-slate-300/50 shadow-sm z-10"
			onclick={copyToClipboard}
			title={isCopied ? 'Copied!' : 'Copy to clipboard'}
		>
			{#if isCopied}
				<Check class="h-3 w-3 text-green-600" />
			{:else}
				<ClipboardCopy class="h-3 w-3 text-slate-600" />
			{/if}
		</Button>{/if}</pre>
</div>
