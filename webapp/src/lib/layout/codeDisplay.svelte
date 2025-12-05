<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import clsx from 'clsx';
	import { Check, ClipboardCopy } from 'lucide-svelte';
	import { codeToHtml, type BundledLanguage, type BundledTheme } from 'shiki';

	import Button from '@/components/ui/button/button.svelte';

	//

	type Props = {
		content: string;
		class?: string;
		hideCopyButton?: boolean;
		language: BundledLanguage;
		theme?: BundledTheme;
		containerClass?: string;
		contentClass?: ClassValue;
	};

	let {
		content,
		class: className = '',
		hideCopyButton = false,
		language,
		containerClass = '',
		theme,
		contentClass = ''
	}: Props = $props();

	//

	let isCopied = $state(false);
	let highlighted = $state('');
	let isDarkTheme = $state(true);

	const actualTheme: BundledTheme = $derived(
		theme || (isDarkTheme ? 'catppuccin-frappe' : 'github-light')
	);

	async function updateHighlighting() {
		const classes = ['p-4 w-0 grow overflow-scroll', clsx(contentClass)];
		// Calculate classes outside `await` and `transformers`
		// so that Svelte's reactivity system can track changes

		highlighted = await codeToHtml(content, {
			lang: language,
			theme: actualTheme,
			transformers: [
				{
					pre(node) {
						this.addClassToHast(node, classes);
					}
				}
			]
		});
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

	// function toggleTheme() {
	// 	isDarkTheme = !isDarkTheme;
	// }

	const preClasses = $derived(
		className || 'border border-slate-200 bg-white p-4 overflow-x-auto text-sm'
	);
</script>

<div class={['relative flex w-full overflow-hidden rounded-md border', containerClass]}>
	{#if highlighted}
		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html highlighted}
		{@render copyButton()}
	{:else}
		<pre
			class={['relative', preClasses]}
			class:language-json={language === 'json'}
			class:language-yaml={language === 'yaml'}>
		{content}
		{@render copyButton()}
	</pre>
	{/if}
</div>

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

			<!-- {#if highlighted}
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
			{/if} -->
		</div>
	{/if}
{/snippet}
