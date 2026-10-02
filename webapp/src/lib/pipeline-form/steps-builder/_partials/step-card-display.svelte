<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Component, Snippet } from 'svelte';

	import { onDestroy, tick, untrack } from 'svelte';
	import { TriangleAlert } from '@lucide/svelte';
	import { Comp } from '$lib/renderable';
	import { showPipelineFormError } from '$pipeline-form/errors.js';
	import { Enrich404Error, type EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
	import * as steps from '$pipeline-form/steps';

	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';

	import {
		bodyMaxHeightWithinCard,
		cancelMotion,
		playInCardEnter,
		playInCardExit,
		type MotionHandle
	} from './in-card-motion.js';
	import { inCardFormHostClass } from '../in-card-edit.js';
	import { getStepData, getStepError } from './index.js';

	//

	type Props = {
		step: EnrichedStep;
		topRight?: Snippet;
		/** When set with `showFormBody`, mounts the in-card form (enter motion crossfades it in). */
		formBody?: Snippet;
		showFormBody?: boolean;
		/**
		 * After enter scroll settles — starts fade-summary → fade-form → grow.
		 * Ignored when not showing the form body.
		 */
		expandReady?: boolean;
		/** Cap for the *whole card* while editing (px). Body grow subtracts header chrome. */
		maxHeightPx?: number;
		/** Becomes true after enter grow finishes; parent uses it for h-full / max-height. */
		enterComplete?: boolean;
		/** Fired after exit shrink/crossfade finishes — clear the held form mode. */
		onExitComplete?: () => void;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		footer?: Snippet | Comp<Component<any>>;
		readonly?: boolean;
		editing?: boolean;
		selected?: boolean;
		hovered?: boolean;
		class?: string;
	};

	let {
		step,
		topRight,
		formBody,
		showFormBody = false,
		expandReady = false,
		maxHeightPx,
		enterComplete = $bindable(false),
		onExitComplete,
		footer,
		readonly = false,
		editing = false,
		selected = false,
		hovered = false,
		class: className
	}: Props = $props();

	const { classes, labels, icon } = $derived(steps.getDisplayData(step[0].use));

	const config = $derived(steps.getConfigByType(step[0].use));
	const stepError = $derived(getStepError(step));
	const stepData = $derived(getStepData(step));
	const cardData = $derived.by(() => {
		if (!stepData) return undefined;
		try {
			return config?.cardData(stepData);
		} catch (e) {
			showPipelineFormError(e);
			return e instanceof Error ? e : new Error(String(e));
		}
	});
	const CardDetailsComponent = $derived(config?.CardDetailsComponent);

	/** Keep summary painted until enter crossfade finishes (or during exit). */
	let showDisplayLayer = $state(true);
	/** Body cap (card max − chrome); kept on the lock so the save bar does not reflow. */
	let bodyMaxPx = $state<number | undefined>(undefined);
	let exiting = $state(false);

	let cardRoot: HTMLElement | null = $state(null);
	let bodyLock: HTMLElement | null = $state(null);
	let displayRoot: HTMLElement | null = $state(null);
	let formHost: HTMLElement | null = $state(null);

	let motion: MotionHandle | undefined;
	let motionToken = 0;

	$effect(() => {
		if (showFormBody) return;
		// Form fully cleared — restore idle display state.
		untrack(() => {
			motionToken++;
			motion?.cancel();
			motion = undefined;
			showDisplayLayer = true;
			enterComplete = false;
			bodyMaxPx = undefined;
			exiting = false;
			cancelMotion(bodyLock);
			cancelMotion(displayRoot);
			cancelMotion(formHost);
		});
	});

	$effect(() => {
		if (!showFormBody || !expandReady || !editing) return;
		if (untrack(() => enterComplete || exiting)) return;
		const card = cardRoot;
		const lock = bodyLock;
		const display = displayRoot;
		const form = formHost;
		if (!card || !lock || !display || !form) return;

		const token = ++motionToken;
		untrack(() => {
			motion?.cancel();
			const bodyMax =
				typeof maxHeightPx === 'number' && Number.isFinite(maxHeightPx)
					? bodyMaxHeightWithinCard(card, display, maxHeightPx)
					: undefined;
			bodyMaxPx = bodyMax;
			motion = playInCardEnter({
				lock,
				display,
				form,
				maxHeightPx: bodyMax,
				onComplete: async () => {
					if (token !== motionToken) return;
					showDisplayLayer = false;
					enterComplete = true;
					// Wait for flex host classes + lock height binding before the
					// motion helper clears absolute fill (avoids save-bar jump).
					await tick();
				}
			});
		});
	});

	// Exit: shrink to summary height, then crossfade — do not collapse to zero.
	$effect(() => {
		if (editing || !showFormBody) return;
		if (untrack(() => exiting)) return;

		const token = ++motionToken;
		untrack(() => {
			motion?.cancel();
			exiting = true;

			if (!enterComplete) {
				// Enter never finished — drop straight back to the summary.
				showDisplayLayer = true;
				enterComplete = false;
				bodyMaxPx = undefined;
				exiting = false;
				onExitComplete?.();
				return;
			}

			showDisplayLayer = true;
			void tick().then(() => {
				if (token !== motionToken) return;
				const lock = bodyLock;
				const display = displayRoot;
				const form = formHost;
				if (!lock || !display || !form) {
					exiting = false;
					onExitComplete?.();
					return;
				}
				motion = playInCardExit({
					lock,
					display,
					form,
					onComplete: () => {
						if (token !== motionToken) return;
						enterComplete = false;
						bodyMaxPx = undefined;
						exiting = false;
						onExitComplete?.();
					}
				});
			});
		});
	});

	onDestroy(() => {
		motionToken++;
		motion?.cancel();
		cancelMotion(bodyLock);
		cancelMotion(displayRoot);
		cancelMotion(formHost);
	});
</script>

<div
	bind:this={cardRoot}
	class={[
		'group flex min-h-0 flex-col overflow-hidden rounded-md border bg-card',
		classes.border,
		!readonly && 'hover:ring',
		editing && 'ring-2 ring-primary',
		selected && !editing && 'ring-1 ring-primary/50',
		hovered && !editing && !selected && 'ring',
		className
	]}
>
	<div class={['h-1 shrink-0', classes?.bg]}></div>

	<div class="flex min-h-0 grow flex-col">
		<div class="flex shrink-0 items-center justify-between py-1 pr-1 pl-3">
			<div class={['flex items-center gap-1', classes.text]}>
				<Icon src={icon} size={12} />
				<p class="text-xs">{labels.singular}</p>
			</div>

			{@render topRight?.()}
		</div>

		{#if showFormBody && formBody}
			<div
				bind:this={bodyLock}
				class={['relative min-h-0', enterComplete || exiting ? 'flex min-h-0 grow flex-col' : '']}
				style:height={enterComplete && bodyMaxPx != null ? `${bodyMaxPx}px` : undefined}
				style:max-height={bodyMaxPx != null ? `${bodyMaxPx}px` : undefined}
				data-testid="in-card-body-lock"
				data-enter-complete={enterComplete}
				data-exiting={exiting}
			>
				{#if showDisplayLayer}
					<div
						bind:this={displayRoot}
						class={enterComplete || exiting
							? 'pointer-events-none absolute top-0 right-0 left-0 w-full'
							: undefined}
						data-testid="in-card-display-body"
					>
						{@render displayDetails()}
						{@render displayFooter()}
					</div>
				{/if}

				<div
					bind:this={formHost}
					class={inCardFormHostClass(enterComplete || exiting)}
					data-testid="in-card-form-host"
					inert={!enterComplete || exiting}
				>
					{@render formBody()}
				</div>
			</div>
		{:else}
			{@render displayDetails()}
		{/if}
	</div>

	{#if !showFormBody}
		{@render displayFooter()}
	{/if}
</div>

{#snippet displayDetails()}
	<div class="space-y-4 p-3 pt-2">
		<div>
			{#if stepError}
				<div class="rounded-md bg-red-700 p-3 text-white">
					<div class="flex items-center gap-2">
						<TriangleAlert size={12} />
						<p class="text-xs">{stepError.message}</p>
					</div>
					{#if stepError instanceof Enrich404Error}
						<p class="pt-2 text-xs opacity-60">{stepError.description}</p>
					{/if}
				</div>
			{:else if cardData instanceof Error}
				<div class="rounded-md bg-red-700 p-3 text-white">
					<div class="flex items-center gap-2">
						<TriangleAlert size={12} />
						<p class="text-xs">{cardData.message}</p>
					</div>
				</div>
			{:else if step[0].use === 'debug'}
				<div class="text-xs text-gray-500">{m.debug_step_description()}</div>
			{:else if cardData}
				{@const { title, copyText, avatar } = cardData}
				<div class="flex items-center gap-3">
					<Avatar src={avatar} fallback={title} class="size-10 rounded-sm border" />
					<div class="space-y-1">
						{#if cardData.beforeTitle}
							<T class="mb-0! text-xs text-muted-foreground">
								{cardData.beforeTitle}
							</T>
						{/if}
						<div class="flex items-center gap-1 leading-snug">
							<p class="text-balance">
								{#if cardData.publicUrl}
									<A href={cardData.publicUrl} target="_blank">
										{title}
									</A>
								{:else}
									{title}
								{/if}

								{#if copyText}
									<CopyButtonSmall
										textToCopy={copyText}
										size="mini"
										class="inline-flex"
									/>
								{/if}
							</p>
						</div>
					</div>
				</div>
			{/if}
		</div>

		{#if cardData && !(cardData instanceof Error) && cardData.meta}
			<div class="space-y-0.5">
				{#each Object.entries(cardData.meta) as [key, value] (key)}
					<p class="text-xs text-muted-foreground">
						<span class="font-medium capitalize">{key}:</span>
						{value}
					</p>
				{/each}
			</div>
		{/if}

		{#if CardDetailsComponent && stepData}
			<CardDetailsComponent data={stepData} />
		{/if}
	</div>
{/snippet}

{#snippet displayFooter()}
	{#if footer instanceof Comp}
		{@const Footer = footer.component}
		<Footer {...footer.props} />
	{:else if footer}
		{@render footer()}
	{/if}
{/snippet}
