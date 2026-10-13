<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Attachment } from 'svelte/attachments';

	import {
		CircleCheckIcon,
		CircleDashedIcon,
		CircleIcon,
		CircleSlashIcon,
		CircleStopIcon,
		CircleXIcon,
		InfoIcon,
		LoaderCircleIcon,
		TriangleAlertIcon
	} from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';

	import { m } from '@/i18n';

	import type { MaestroFlowView, MaestroRowView } from './maestro-timeline';

	type Props = {
		flows: MaestroFlowView[];
		/** Shown instead of the rows when there are no flows. */
		empty?: string;
		highlight?: { flowId: string; index: number };
		onSelect?: (row: MaestroRowView) => void;
	};

	let { flows, empty, highlight, onSelect }: Props = $props();

	const STATUS_ICONS: Record<string, IconComponent> = {
		RUNNING: LoaderCircleIcon,
		COMPLETED: CircleCheckIcon,
		FAILED: CircleXIcon,
		WARNED: TriangleAlertIcon,
		SKIPPED: CircleSlashIcon,
		PENDING: CircleDashedIcon,
		INTERRUPTED: CircleStopIcon
	};

	function rowIcon(row: MaestroRowView): IconComponent {
		if (row.kind === 'notice') return InfoIcon;
		return Object.hasOwn(STATUS_ICONS, row.status) ? STATUS_ICONS[row.status] : CircleIcon;
	}

	const scrollIntoView: Attachment<HTMLElement> = (element) => {
		element.scrollIntoView({ block: 'center' });
	};
</script>

<section class="box" aria-labelledby="wallet-actions-title">
	<h2 id="wallet-actions-title">{m.Wallet_actions()}</h2>
	<div class="log">
		{#if flows.length === 0 && empty}
			<p class="empty">{empty}</p>
		{/if}
		{#each flows as flow (flow.flowId)}
			<p class="flow">{flow.title}</p>
			<ol>
				{#each flow.rows as row (row.index)}
					{@const highlighted =
						highlight?.flowId === flow.flowId && highlight.index === row.index}
					<li
						data-status={row.status}
						data-kind={row.kind}
						class:highlighted
						style:padding-left="{row.depth * 1.25}em"
						{@attach highlighted && scrollIntoView}
					>
						{#if onSelect}
							<button type="button" class="row" onclick={() => onSelect(row)}>
								{@render rowContent(row)}
							</button>
						{:else}
							<div class="row">
								{@render rowContent(row)}
							</div>
						{/if}
					</li>
				{/each}
			</ol>
		{/each}
	</div>
</section>

{#snippet rowContent(row: MaestroRowView)}
	{@const Icon = rowIcon(row)}
	<Icon class={Icon === CircleIcon ? 'unknown' : undefined} />
	<div class="text">
		<span>{row.description}</span>
		{#if row.message}
			<p class="reason">{row.message}</p>
		{/if}
	</div>
{/snippet}

<style>
	.box {
		display: flex;
		flex-direction: column;
		height: 100%;
		border: 1px solid #1e293b;
		border-radius: 8px;
		background: #0b1020;
		color: #e2e8f0;
		overflow: hidden;
	}
	h2 {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 0;
		padding: 8px 12px;
		border-bottom: 1px solid #1e293b;
		background: #111827;
		color: #94a3b8;
		font-size: 12px;
		font-weight: 600;
	}
	h2::before {
		content: '';
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #3d1fc4;
	}
	.log {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		padding: 10px 12px;
		font:
			12px/1.5 ui-monospace,
			SFMono-Regular,
			Menlo,
			Consolas,
			monospace;
	}
	.empty {
		margin: 0;
		color: #64748b;
	}
	.flow {
		margin: 0 0 4px;
		color: #94a3b8;
		font-weight: 700;
	}
	ol {
		margin: 0 0 12px;
		padding: 0;
		list-style: none;
	}
	li {
		padding-top: 1px;
		padding-bottom: 1px;
		border-radius: 4px;
	}
	/* Drawn inside the row: rows are only 2px apart, so an outer outline would
	   cover the neighbouring lines. */
	li.highlighted {
		outline: 1px solid #f87171;
		outline-offset: -1px;
		background: rgb(248 113 113 / 0.08);
	}
	.row {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		width: 100%;
		padding: 0;
		border: 0;
		background: none;
		color: inherit;
		font: inherit;
		text-align: left;
	}
	button.row {
		cursor: pointer;
	}
	button.row:hover {
		background: #111827;
	}
	.text {
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.reason {
		margin: 1px 0 2px;
		color: #fca5a5;
		font-size: 11px;
	}
	.row :global(svg) {
		flex: none;
		width: 14px;
		height: 14px;
		margin-top: 2px;
		color: #94a3b8;
	}
	.row :global(svg.unknown) {
		width: 8px;
		height: 8px;
		margin: 5px 3px 0;
	}
	li[data-status='RUNNING'] .row :global(svg) {
		color: #a5b4fc;
		animation: spin 1s linear infinite;
	}
	li[data-status='COMPLETED'] .row :global(svg) {
		color: #4ade80;
	}
	li[data-status='FAILED'] .row :global(svg) {
		color: #f87171;
	}
	li[data-status='FAILED'] span {
		color: #fecaca;
	}
	li[data-status='WARNED'] .row :global(svg) {
		color: #fbbf24;
	}
	li[data-status='WARNED'] .reason {
		color: #fcd34d;
	}
	li[data-status='SKIPPED'] .row :global(svg),
	li[data-status='PENDING'] .row :global(svg) {
		color: #64748b;
	}
	li[data-status='SKIPPED'] span,
	li[data-status='INTERRUPTED'] span {
		color: #94a3b8;
	}
	li[data-kind='flow'] span {
		color: #fca5a5;
		font-weight: 700;
	}
	li[data-kind='notice'] span {
		color: #94a3b8;
		font-style: italic;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
