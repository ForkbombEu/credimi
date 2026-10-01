<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Attachment } from 'svelte/attachments';
	import type { Readable } from 'svelte/store';

	import { Check, ClipboardCopy } from '@lucide/svelte';
	import type { SvelteVirtualizer } from '@tanstack/svelte-virtual';

	import Button from '@/components/ui/button/button.svelte';

	import type { ActiveUnit } from '../scroll-follow/active-unit.js';
	import type { YamlPreviewParts } from '../yaml-preview-split.js';
	import type { YamlStepsVirtualizer } from '../yaml-virtualizer.svelte.js';

	import YamlShikiBlock from './yaml-shiki-block.svelte';

	type Props = {
		/** Full document — SoT for copy; fragments come from `parts`. */
		yaml: string;
		parts: YamlPreviewParts;
		yamlVirtualizer: YamlStepsVirtualizer;
		/** Store auto-subscribe target — `$yamlVirt` in markup. */
		yamlVirt: Readable<SvelteVirtualizer<HTMLElement, Element>>;
		scrollMargin: number;
		/** YAML column scroller — used to measure scrollMargin for the virtual step list. */
		scrollContainer?: HTMLElement | null;
		isUnitSelected: (section: ActiveUnit['section'], index: number) => boolean;
		isUnitHovered: (section: ActiveUnit['section'], index: number) => boolean;
		onUnitClick: (unit: ActiveUnit) => void;
		onUnitHover: (unit: ActiveUnit | null) => void;
		/** Reports offset of the virtual step list within the YAML scroller. */
		onHeaderHeightChange?: (height: number) => void;
		endPadPx?: number;
	};

	let {
		yaml,
		parts,
		yamlVirtualizer,
		yamlVirt,
		scrollMargin,
		scrollContainer = null,
		isUnitSelected,
		isUnitHovered,
		onUnitClick,
		onUnitHover,
		onHeaderHeightChange,
		endPadPx = 0
	}: Props = $props();

	let isCopied = $state(false);
	/** Wraps everything above the virtual step list (padding + header). */
	let beforeStepsEl: HTMLElement | null = $state(null);
	let virtualListEl: HTMLElement | null = $state(null);

	const measureStepBlock: Attachment = (node) => {
		yamlVirtualizer.measureElement(node);
	};

	$effect(() => {
		const list = virtualListEl;
		const scroller = scrollContainer;
		const before = beforeStepsEl;
		if (!list || !scroller || !onHeaderHeightChange) return;
		const update = () => {
			const listRect = list.getBoundingClientRect();
			const scrollRect = scroller.getBoundingClientRect();
			onHeaderHeightChange(Math.max(0, listRect.top - scrollRect.top + scroller.scrollTop));
		};
		update();
		const ro = new ResizeObserver(update);
		ro.observe(list);
		if (before) ro.observe(before);
		ro.observe(scroller);
		return () => {
			ro.disconnect();
		};
	});

	async function copyToClipboard() {
		if (!yaml) return;
		try {
			await navigator.clipboard.writeText(yaml);
			isCopied = true;
			setTimeout(() => {
				isCopied = false;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy text: ', err);
		}
	}
</script>

<div class="relative flex min-h-0 w-full flex-col bg-[#303446] text-sm text-white">
	{#if yaml}
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

	<div class="min-w-0 p-4 pt-3">
		<!-- Header: name / runtime / steps: — not inside the virtual step list -->
		<div bind:this={beforeStepsEl}>
			{#if parts.header}
				<YamlShikiBlock content={parts.header} />
			{/if}
		</div>

		<!-- Virtual per-step Shiki blocks (index-aligned with cards) -->
		{#if parts.steps.length > 0}
			<div
				bind:this={virtualListEl}
				class="relative w-full"
				style:height="{$yamlVirt.getTotalSize()}px"
			>
				{#each $yamlVirt.getVirtualItems() as vItem (parts.steps[vItem.index]?.text ?? vItem.key)}
					{@const block = parts.steps[vItem.index]}
					{@const index = vItem.index}
					{#if block}
						<div
							{@attach measureStepBlock}
							data-index={index}
							data-yaml-section="steps"
							data-yaml-index={index}
							class="absolute right-0 left-0 w-full min-w-full"
							style:top="{vItem.start - scrollMargin}px"
							role="group"
							tabindex="-1"
							onmouseenter={() => onUnitHover({ section: 'steps', index })}
							onmouseleave={() => onUnitHover(null)}
						>
							<YamlShikiBlock
								content={block.text}
								interactive
								selected={isUnitSelected('steps', index)}
								hovered={isUnitHovered('steps', index)}
								onclick={() => onUnitClick({ section: 'steps', index })}
							/>
						</div>
					{/if}
				{/each}
			</div>
		{/if}

		<!-- Follow-ups: separate section under the virtual step list -->
		{#if parts.followUpsPreamble}
			<YamlShikiBlock content={parts.followUpsPreamble} />
		{/if}

		{#each parts.followUps as followUp (followUp.index)}
			{@const index = followUp.index}
			<div
				data-yaml-section="follow-ups"
				data-yaml-index={index}
				role="group"
				tabindex="-1"
				onmouseenter={() => onUnitHover({ section: 'follow-ups', index })}
				onmouseleave={() => onUnitHover(null)}
			>
				<YamlShikiBlock
					content={followUp.text}
					interactive
					selected={isUnitSelected('follow-ups', index)}
					hovered={isUnitHovered('follow-ups', index)}
					onclick={() => onUnitClick({ section: 'follow-ups', index })}
				/>
			</div>
			{#if parts.betweenFollowUps[index]}
				<YamlShikiBlock content={parts.betweenFollowUps[index]!} />
			{/if}
		{/each}

		{#if endPadPx > 0}
			<div
				class="pointer-events-none shrink-0"
				style:height="{endPadPx}px"
				aria-hidden="true"
			></div>
		{/if}
	</div>
</div>
