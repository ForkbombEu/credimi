<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import { Check, ClipboardCopy } from '@lucide/svelte';
	import clsx from 'clsx';
	import { codeToHtml, type BundledLanguage, type BundledTheme } from 'shiki';

	import Button from '@/components/ui/button/button.svelte';

	//

	type LineRange = {
		/** Inclusive 0-based */
		start: number;
		/** Inclusive 0-based */
		end: number;
	};

	type Props = {
		content: string;
		class?: string;
		hideCopyButton?: boolean;
		language: BundledLanguage;
		theme?: BundledTheme;
		containerClass?: string;
		contentClass?: ClassValue;
		/** Soft wash over these lines (0-based inclusive). */
		highlightLines?: LineRange | null;
		onLineClick?: (line: number) => void;
		scroller?: HTMLElement | null;
	};

	let {
		content,
		class: className = '',
		hideCopyButton = false,
		language,
		containerClass = '',
		theme,
		contentClass = '',
		highlightLines = null,
		onLineClick,
		scroller = $bindable<HTMLElement | null>(null)
	}: Props = $props();

	//

	let isCopied = $state(false);
	let highlighted = $state('');
	let isDarkTheme = $state(true);
	let containerEl: HTMLDivElement | undefined = $state();

	const actualTheme: BundledTheme = $derived(
		theme || (isDarkTheme ? 'catppuccin-frappe' : 'github-light')
	);

	async function updateHighlighting() {
		const classes = ['p-4 w-0 grow overflow-scroll', clsx(contentClass)];
		const wash = highlightLines;

		highlighted = await codeToHtml(content, {
			lang: language,
			theme: actualTheme,
			transformers: [
				{
					pre(node) {
						this.addClassToHast(node, classes);
					},
					line(node, line) {
						const idx = line - 1;
						node.properties = {
							...node.properties,
							'data-line': String(idx)
						};
						this.addClassToHast(node, 'code-display-line');
						if (wash && idx >= wash.start && idx <= wash.end) {
							this.addClassToHast(node, 'code-display-line-wash');
						}
					}
				}
			]
		});
	}

	$effect(() => {
		void updateHighlighting();
	});

	$effect(() => {
		const html = highlighted;
		const pre = html ? (containerEl?.querySelector('pre') ?? null) : null;
		scroller = pre;

		if (!pre || !onLineClick) return;

		const handler = (event: MouseEvent) => {
			const target = event.target;
			if (!(target instanceof Element)) return;
			const lineEl = target.closest('[data-line]');
			if (!lineEl) return;
			const line = Number(lineEl.getAttribute('data-line'));
			if (!Number.isInteger(line)) return;
			onLineClick(line);
		};

		pre.addEventListener('click', handler);
		return () => {
			pre.removeEventListener('click', handler);
		};
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

	const preClasses = $derived(
		className || 'border border-slate-200 bg-white p-4 overflow-x-auto text-sm'
	);
</script>

<div
	bind:this={containerEl}
	class={['relative flex w-full overflow-hidden rounded-md border', containerClass]}
	class:code-display-clickable={Boolean(onLineClick)}
>
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
		<div class="absolute top-2 right-2 z-10 flex flex-col gap-1">
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
		</div>
	{/if}
{/snippet}

<style>
	:global(.code-display-line-wash) {
		background-color: color-mix(in oklab, var(--brand-primary, #3730a3) 16%, transparent);
		display: inline-block;
		width: 100%;
	}

	:global(.code-display-clickable .code-display-line) {
		cursor: pointer;
	}
</style>
