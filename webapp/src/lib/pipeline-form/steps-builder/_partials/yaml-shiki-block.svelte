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
			selected && 'yaml-preview-block-selected ring-2 ring-orange-500',
			hovered && !selected && 'yaml-preview-block-hovered ring-2 ring-orange-500/60',
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
				class="yaml-preview-block-pre m-0 w-full min-w-full overflow-x-auto border-0 bg-transparent text-sm">{content}</pre>
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
		/* Thin bar; thumb hidden until hover/focus so Windows does not paint a fat classic bar on every chunk. */
		scrollbar-width: thin;
		scrollbar-color: transparent transparent;
	}

	:global(.yaml-preview-block-pre::-webkit-scrollbar) {
		height: 6px;
	}

	:global(.yaml-preview-block-pre::-webkit-scrollbar-track) {
		background: transparent;
	}

	:global(.yaml-preview-block-pre::-webkit-scrollbar-thumb) {
		background: transparent;
		border-radius: 3px;
	}

	@media (hover: hover) {
		:global(.yaml-preview-block:hover .yaml-preview-block-pre),
		:global(.yaml-preview-block:focus-within .yaml-preview-block-pre) {
			scrollbar-color: rgb(255 255 255 / 0.25) transparent;
		}

		:global(.yaml-preview-block:hover .yaml-preview-block-pre::-webkit-scrollbar-thumb),
		:global(.yaml-preview-block:focus-within .yaml-preview-block-pre::-webkit-scrollbar-thumb) {
			background: rgb(255 255 255 / 0.25);
		}
	}

	/* Touch / no-hover: keep a quiet thin thumb so overflow stays discoverable. */
	@media (hover: none) {
		:global(.yaml-preview-block-pre) {
			scrollbar-color: rgb(255 255 255 / 0.25) transparent;
		}

		:global(.yaml-preview-block-pre::-webkit-scrollbar-thumb) {
			background: rgb(255 255 255 / 0.25);
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
		background-color: rgb(255 255 255 / 0.15);
	}

	:global(.yaml-preview-block-selected .code-display-line) {
		background-color: rgb(255 255 255 / 0.22);
	}
</style>
