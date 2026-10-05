<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { BundledTheme } from 'shiki';

	import { highlightYamlFragment } from '../yaml-shiki-cache.js';

	type Props = {
		content: string;
		/** When true, shows pointer cursor (click handled by parent wrapper). */
		interactive?: boolean;
		selected?: boolean;
		hovered?: boolean;
		theme?: BundledTheme;
		onclick?: () => void;
		class?: string;
	};

	let {
		content,
		interactive = false,
		selected = false,
		hovered = false,
		theme = 'catppuccin-frappe',
		onclick,
		class: className = ''
	}: Props = $props();

	let highlighted = $state('');
	let highlightGeneration = 0;

	$effect(() => {
		const text = content;
		const activeTheme = theme;
		const generation = ++highlightGeneration;
		if (!text) {
			highlighted = '';
			return;
		}
		void highlightYamlFragment(text, activeTheme).then((html) => {
			if (generation !== highlightGeneration) return;
			highlighted = html;
		});
	});
</script>

{#if content}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class={[
			'yaml-preview-block relative w-full min-w-full rounded-sm',
			interactive && 'yaml-preview-block-interactive',
			selected && 'yaml-preview-block-selected ring-1 ring-orange-400',
			hovered && !selected && 'yaml-preview-block-hovered ring-1 ring-white/40',
			(hovered || selected) && 'overflow-hidden',
			className
		]}
		{onclick}
	>
		{#if highlighted}
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html highlighted}
		{:else}
			<pre
				class="yaml-preview-block-pre scrollbar-on-dark m-0 w-full min-w-full overflow-x-auto border-0 bg-transparent text-sm">{content}</pre>
		{/if}
	</div>
{/if}

<style>
	/*
	  Pane owns the dark canvas. Shiki theme backgrounds are stripped so every
	  virtual row paints the same full-width surface (avoids mid-scroll gutters
	  when a block's content is narrower than its siblings).
	*/
	:global(.yaml-preview-block-pre) {
		width: 100%;
		min-width: 100%;
		box-sizing: border-box;
		background-color: transparent !important;
	}

	/*
	  Per-chunk horizontal bars: inherit .scrollbar-on-dark tokens, but keep the
	  thumb quiet until hover/focus (Windows otherwise shows a bar on every step).
	*/
	@media (hover: hover) {
		:global(.yaml-preview-block-pre) {
			--scrollbar-thumb: transparent;
		}

		:global(.yaml-preview-block:hover .yaml-preview-block-pre),
		:global(.yaml-preview-block:focus-within .yaml-preview-block-pre) {
			--scrollbar-thumb: rgb(255 255 255 / 0.35);
		}
	}

	:global(.yaml-preview-block-pre > code) {
		display: flex;
		flex-direction: column;
		min-width: 100%;
		width: max-content;
		box-sizing: border-box;
	}

	:global(.yaml-preview-block .code-display-line) {
		display: block;
		width: 100%;
		min-height: 1.25em;
		box-sizing: border-box;
	}

	:global(.yaml-preview-block-interactive) {
		cursor: pointer;
	}

	/*
	  Wash sits on the lines (like CodeDisplay), not behind Shiki's opaque pre —
	  otherwise hover is invisible. Soft 15% hover; selected slightly stronger.
	*/
	:global(.yaml-preview-block-hovered .code-display-line) {
		background-color: rgb(255 255 255 / 0.05);
	}

	:global(.yaml-preview-block-selected .code-display-line) {
		background-color: rgb(255 255 255 / 0.05);
	}
</style>
