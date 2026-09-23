<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Attachment } from 'svelte/attachments';
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
		/** Stronger wash over these lines (0-based inclusive). */
		selectedLines?: LineRange | null;
		/** Lighter wash over these lines (0-based inclusive). Selected wins when both apply. */
		hoverLines?: LineRange | null;
		onLineClick?: (line: number) => void;
		onLineHover?: (line: number | null) => void;
		/** Fraction of scroller height as bottom padding so the last block can center. */
		endPadRatio?: number;
		scroller?: HTMLElement | null;
		/** Optional attachment for the Shiki `<pre>` scroller (e.g. peer scroll-follow). */
		scrollerAttach?: Attachment;
	};

	let {
		content,
		class: className = '',
		hideCopyButton = false,
		language,
		containerClass = '',
		theme,
		contentClass = '',
		selectedLines = null,
		hoverLines = null,
		onLineClick,
		onLineHover,
		endPadRatio = 0,
		scroller = $bindable<HTMLElement | null>(null),
		scrollerAttach
	}: Props = $props();

	//

	let isCopied = $state(false);
	let highlighted = $state('');
	let isDarkTheme = $state(true);
	let containerEl: HTMLDivElement | undefined = $state();
	let highlightGeneration = 0;
	let lastHoverLine: number | null = null;

	const actualTheme: BundledTheme = $derived(
		theme || (isDarkTheme ? 'catppuccin-frappe' : 'github-light')
	);

	async function updateHighlighting() {
		const generation = ++highlightGeneration;
		const classes = [
			'p-4 w-full min-h-0 grow h-full overflow-auto code-display-scroller',
			clsx(contentClass)
		];

		const html = await codeToHtml(content, {
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
						// Empty source lines are empty spans; flex would collapse them.
						// NBSP keeps intentional blank rows one line tall.
						if (!node.children || node.children.length === 0) {
							node.children = [{ type: 'text', value: '\u00A0' }];
						}
					}
				}
			]
		});

		if (generation !== highlightGeneration) return;
		highlighted = html;
	}

	function inRange(idx: number, wash: LineRange | null): boolean {
		return wash != null && idx >= wash.start && idx <= wash.end;
	}

	function applyWashClasses(
		pre: HTMLElement,
		selected: LineRange | null,
		hover: LineRange | null
	) {
		for (const el of pre.querySelectorAll<HTMLElement>('[data-line]')) {
			const idx = Number(el.getAttribute('data-line'));
			if (!Number.isInteger(idx)) continue;
			const isSelected = inRange(idx, selected);
			el.classList.toggle('code-display-line-wash-selected', isSelected);
			el.classList.toggle('code-display-line-wash-hover', !isSelected && inRange(idx, hover));
		}
	}

	function lineFromEvent(event: MouseEvent): number | null {
		const target = event.target;
		if (!(target instanceof Element)) return null;
		const lineEl = target.closest('[data-line]');
		if (!lineEl) return null;
		const line = Number(lineEl.getAttribute('data-line'));
		return Number.isInteger(line) ? line : null;
	}

	/** Shiki `<pre>` from `{@html}`; null when unhighlighted or not yet in the DOM. */
	function findHighlightedPre(): HTMLElement | null {
		if (!highlighted) return null;
		return containerEl?.querySelector('pre') ?? null;
	}

	// Shiki: content/language/theme/contentClass only — never wash props.
	$effect(() => {
		void content;
		void language;
		void actualTheme;
		void contentClass;
		void updateHighlighting();
	});

	// Attach stays stable across wash updates; `{@html}` blocks `{@attach}` on the real `<pre>`.
	$effect(() => {
		const pre = findHighlightedPre();
		scroller = pre;

		if (!pre) return;

		const cleanups: Array<() => void> = [];

		if (scrollerAttach) {
			const detach = scrollerAttach(pre);
			if (typeof detach === 'function') cleanups.push(detach);
		}

		if (onLineClick) {
			const onClick = (event: MouseEvent) => {
				const line = lineFromEvent(event);
				if (line === null) return;
				onLineClick(line);
			};
			pre.addEventListener('click', onClick);
			cleanups.push(() => pre.removeEventListener('click', onClick));
		}

		if (onLineHover) {
			const emitHover = (line: number | null) => {
				if (line === lastHoverLine) return;
				lastHoverLine = line;
				onLineHover(line);
			};

			const onMove = (event: MouseEvent) => {
				emitHover(lineFromEvent(event));
			};

			const onLeave = () => {
				emitHover(null);
			};

			pre.addEventListener('mousemove', onMove);
			pre.addEventListener('mouseover', onMove);
			pre.addEventListener('mouseleave', onLeave);
			cleanups.push(() => {
				pre.removeEventListener('mousemove', onMove);
				pre.removeEventListener('mouseover', onMove);
				pre.removeEventListener('mouseleave', onLeave);
				if (lastHoverLine !== null) {
					lastHoverLine = null;
					onLineHover(null);
				}
			});
		}

		if (cleanups.length === 0) return;
		return () => {
			for (const cleanup of cleanups) cleanup();
		};
	});

	// Wash is frequent (hover); keep separate so attach/listeners stay stable.
	$effect(() => {
		const selected = selectedLines;
		const hover = hoverLines;
		const pre = scroller;
		if (!(pre instanceof HTMLElement)) return;
		applyWashClasses(pre, selected, hover);
	});

	// Bottom pad so the last block can scroll to viewport center.
	$effect(() => {
		const ratio = endPadRatio;
		// Prefer bindable scroller; fallback covers the unhighlighted plain `<pre>` branch.
		const pre = scroller ?? containerEl?.querySelector('pre');
		if (!pre || !(ratio > 0)) {
			if (pre) pre.style.paddingBottom = '';
			return;
		}
		const update = () => {
			// clientHeight includes padding — subtract current pad to avoid feedback growth.
			const currentPad = parseFloat(getComputedStyle(pre).paddingBottom) || 0;
			const viewH = Math.max(0, pre.clientHeight - currentPad);
			pre.style.paddingBottom = `${Math.round(viewH * ratio)}px`;
		};
		update();
		const ro = new ResizeObserver(update);
		ro.observe(pre);
		return () => {
			ro.disconnect();
			pre.style.paddingBottom = '';
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
	class={[
		'relative flex h-full min-h-0 w-full flex-col overflow-hidden rounded-md border',
		containerClass
	]}
	class:code-display-clickable={Boolean(onLineClick || onLineHover)}
>
	{#if highlighted}
		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html highlighted}
		{@render copyButton()}
	{:else}
		<pre
			class={['relative min-h-0 grow overflow-auto', preClasses]}
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
	/*
	  Shiki puts a literal `\n` between each `.line` span. With `display: block` on
	  those spans, the newlines become extra anonymous line boxes — a blank-looking
	  gap between every row. Flex/grid on `code` ignores that inter-element whitespace.
	  Empty lines stay one row tall via an NBSP injected in the line transformer.
	  min-width/width:max-content lets the wash span the full scroll width of long lines.
	*/
	:global(.code-display-scroller > code) {
		display: flex;
		flex-direction: column;
		min-width: 100%;
		width: max-content;
	}

	:global(.code-display-line) {
		display: block;
		width: 100%;
		min-height: 1.25em;
		box-sizing: border-box;
	}

	:global(.code-display-line-wash-hover) {
		background-color: rgb(255 255 255 / 0.08);
	}

	:global(.code-display-line-wash-selected) {
		background-color: rgb(255 255 255 / 0.16);
	}

	:global(.code-display-clickable .code-display-line) {
		cursor: pointer;
	}
</style>
