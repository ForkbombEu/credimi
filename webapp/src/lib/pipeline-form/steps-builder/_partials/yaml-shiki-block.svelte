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
			'yaml-preview-block relative w-full',
			interactive && 'yaml-preview-block-interactive',
			selected && 'yaml-preview-block-selected',
			hovered && !selected && 'yaml-preview-block-hovered',
			className
		]}
		{onclick}
	>
		{#if highlighted}
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html highlighted}
		{:else}
			<pre
				class="yaml-preview-block-pre m-0 w-full overflow-x-auto border-0 text-sm">{content}</pre>
		{/if}
	</div>
{/if}

<style>
	:global(.yaml-preview-block-pre > code) {
		display: flex;
		flex-direction: column;
		min-width: 100%;
		width: max-content;
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

	:global(.yaml-preview-block-hovered) {
		background-color: rgb(255 255 255 / 0.08);
	}

	:global(.yaml-preview-block-selected) {
		background-color: rgb(255 255 255 / 0.16);
	}
</style>
