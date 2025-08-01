<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ClipboardCopy, Check, Sun, Moon } from 'lucide-svelte';
	import Button from '@/components/ui/button/button.svelte';
	import { onMount } from 'svelte';
	import { codeToHtml } from 'shiki';

	type Props = {
		content: string;
		class?: string;
		hideCopyButton?: boolean;
		language?: string;
		containerClass?: string;
	};

	let {
		content,
		class: className = '',
		hideCopyButton = false,
		language = '',
		containerClass = ''
	}: Props = $props();

	let isCopied = $state(false);
	let highlighted = $state('');
	let isDarkTheme = $state(true);

	onMount(async () => {
		await updateHighlighting();
	});

	let supportedLanguages: string[] = [];

	onMount(async () => {
		const highlighter = await getHighlighter({ theme: 'nord' });
		supportedLanguages = highlighter.getLoadedLanguages();
		await updateHighlighting();
	});

	async function updateHighlighting() {
		if (content && supportedLanguages.includes(language)) {
			highlighted = await codeToHtml(content, {
				lang: language,
				theme: isDarkTheme ? 'vitesse-dark' : 'github-light',
				transformers: [
					{
						pre(node) {
							this.addClassToHast(node, 'p-4');
						}
					}
				]
			});
		}
	}

	$effect(() => {
		updateHighlighting();
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

	function toggleTheme() {
		isDarkTheme = !isDarkTheme;
	}

	const preClasses = $derived(
		className || 'rounded-lg border border-slate-200 bg-white p-4 overflow-x-auto text-sm'
	);
</script>

{#snippet copyButton()}
	{#if !hideCopyButton && content}
		<div class="absolute right-2 top-2 z-10 flex flex-col gap-1">
			<Button
				type="button"
				variant="ghost"
				size="sm"
				class="h-6 w-6 border border-slate-300/50 bg-white/90 p-0 opacity-80 shadow-sm backdrop-blur-sm hover:bg-white/100 hover:opacity-100"
				onclick={copyToClipboard}
				title={isCopied ? 'Copied!' : 'Copy to clipboard'}
			>
				{#if isCopied}
					<Check class="h-3 w-3 text-green-600" />
				{:else}
					<ClipboardCopy class="h-3 w-3 text-slate-600" size={16} />
				{/if}
			</Button>

			{#if highlighted}
				<Button
					type="button"
					variant="ghost"
					size="sm"
					class="h-6 w-6 border border-slate-300/50 bg-white/90 p-0 opacity-80 shadow-sm backdrop-blur-sm hover:bg-white/100 hover:opacity-100"
					onclick={toggleTheme}
					title={isDarkTheme ? 'Switch to light theme' : 'Switch to dark theme'}
				>
					{#if isDarkTheme}
						<Sun class="h-3 w-3 text-slate-600" />
					{:else}
						<Moon class="h-3 w-3 text-slate-600" />
					{/if}
				</Button>
			{/if}
		</div>
	{/if}
{/snippet}

<div class="relative {containerClass}">
	{#if highlighted}
		<div
			class={preClasses}
			style="padding: 0; margin: 0; overflow: hidden; position: relative;"
		>
			{@html highlighted}
			{@render copyButton()}
		</div>
	{:else}
		<pre
			class="{preClasses} relative"
			class:language-json={language === 'json'}
			class:language-yaml={language === 'yaml'}>
		{content}
		{@render copyButton()}
	</pre>
	{/if}
</div>
