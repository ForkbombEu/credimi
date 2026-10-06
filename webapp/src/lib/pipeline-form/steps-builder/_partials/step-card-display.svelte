<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Component, Snippet } from 'svelte';

	import { HelpCircle, TriangleAlert, XIcon } from '@lucide/svelte';
	import { Comp } from '$lib/renderable';
	import { showPipelineFormError } from '$pipeline-form/errors.js';
	import { Enrich404Error, type EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
	import * as steps from '$pipeline-form/steps';
	import { onDestroy, tick, untrack } from 'svelte';

	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n/index.js';

	import {
		inCardFormHostClass,
		playInCardEnterLayout,
		playInCardExitLayout
	} from '../in-card-layout.js';
	import InCardFormShell from './in-card-form-shell.svelte';
	import { cancelMotion, type MotionHandle } from './in-card-motion.js';
	import { getStepData, getStepError } from './index.js';

	//

	type Props = {
		step: EnrichedStep;
		topRight?: Snippet;
		/** When set with `showFormBody`, mounts the in-card form (enter motion crossfades it in). */
		formBody?: Snippet;
		showFormBody?: boolean;
		/**
		 * Enter-ready after start-align settles (`editing && phase === 'still'`).
		 * Starts fade-summary → fade-form → grow. Ignored when not showing the form body.
		 */
		expandReady?: boolean;
		/** Cap for the *whole card* while editing (px). Body grow subtracts header chrome. */
		maxHeightPx?: number;
		/** Sibling of the card being edited in place: dimmed and non-interactive (except the pencil). */
		faded?: boolean;
		/**
		 * When set (Composer idle chrome), fade action buttons until hover.
		 * Hub omits this so `topRight` is not faded.
		 */
		actionsDisabled?: boolean;
		docsUrl?: string;
		canSave?: boolean;
		onSave?: () => void;
		onDismiss?: () => void;
		/** Fired after exit shrink/crossfade finishes — clear the held form mode. */
		onExitComplete?: () => void;
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
		faded = false,
		actionsDisabled,
		docsUrl,
		canSave = false,
		onSave,
		onDismiss,
		onExitComplete,
		footer,
		readonly = false,
		editing = false,
		selected = false,
		hovered = false,
		class: className
	}: Props = $props();

	/** True after enter grow finishes; layout/inert only — not bound by parents. */
	let enterComplete = $state(false);

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
	/** Set while exit layout handle is live — form stays inert; not session `exiting`. */
	let exitMotion = $state<MotionHandle | undefined>(undefined);

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
			exitMotion?.cancel();
			exitMotion = undefined;
			showDisplayLayer = true;
			enterComplete = false;
			bodyMaxPx = undefined;
			cancelMotion(bodyLock);
			cancelMotion(displayRoot);
			cancelMotion(formHost);
		});
	});

	// Hide the form host until layout owns it — avoids a visible flash between
	// showFormBody mount and enter-ready (enter previously set these classes itself).
	$effect(() => {
		const form = formHost;
		if (!form || !showFormBody) return;
		if (enterComplete || exitMotion) return;
		form.className = inCardFormHostClass(false);
	});

	$effect(() => {
		if (!showFormBody || !expandReady || !editing) return;
		if (untrack(() => enterComplete || exitMotion)) return;
		const card = cardRoot;
		const lock = bodyLock;
		const display = displayRoot;
		const form = formHost;
		if (!card || !lock || !display || !form) return;

		const token = ++motionToken;
		untrack(() => {
			motion?.cancel();
			motion = playInCardEnterLayout({
				lock,
				display,
				form,
				card,
				cardFillMaxPx: maxHeightPx,
				onSettled: async ({ bodyMaxPx: settledBodyMax }) => {
					if (token !== motionToken) return;
					showDisplayLayer = false;
					enterComplete = true;
					bodyMaxPx = settledBodyMax;
				}
			});
		});
	});

	// Exit: shrink to summary height, then crossfade — do not collapse to zero.
	$effect(() => {
		if (editing || !showFormBody) return;
		if (untrack(() => exitMotion)) return;

		const token = ++motionToken;
		untrack(() => {
			motion?.cancel();

			if (!enterComplete) {
				showDisplayLayer = true;
				enterComplete = false;
				bodyMaxPx = undefined;
				exitMotion = playInCardExitLayout({
					enterSettled: false,
					onComplete: () => {
						if (token !== motionToken) return;
						exitMotion = undefined;
						motion = undefined;
						onExitComplete?.();
					}
				});
				motion = exitMotion;
				return;
			}

			showDisplayLayer = true;
			void tick().then(() => {
				if (token !== motionToken) return;
				const lock = bodyLock;
				const display = displayRoot;
				const form = formHost;
				exitMotion = playInCardExitLayout({
					lock: lock ?? undefined,
					display: display ?? undefined,
					form: form ?? undefined,
					enterSettled: Boolean(lock && display && form),
					onComplete: () => {
						if (token !== motionToken) return;
						enterComplete = false;
						bodyMaxPx = undefined;
						exitMotion = undefined;
						motion = undefined;
						onExitComplete?.();
					}
				});
				motion = exitMotion;
			});
		});
	});

	onDestroy(() => {
		motionToken++;
		motion?.cancel();
		exitMotion?.cancel();
		cancelMotion(bodyLock);
		cancelMotion(displayRoot);
		cancelMotion(formHost);
	});
</script>

<div
	class={[
		'flex min-h-0 flex-col transition-opacity duration-200',
		faded && 'pointer-events-none opacity-40'
	]}
	style:max-height={showFormBody && maxHeightPx != null ? `${maxHeightPx}px` : undefined}
>
	<div
		bind:this={cardRoot}
		class={[
			'group flex min-h-0 flex-col overflow-hidden rounded-md border bg-card',
			classes.border,
			!readonly &&
				!selected &&
				!editing &&
				'hover:border-primary hover:ring-1 hover:ring-primary',
			(editing || selected) && 'border-orange-600 ring-1 ring-orange-600',
			hovered && !editing && !selected && 'border-primary ring-1 ring-primary',
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

				{@render headerChrome()}
			</div>

			{#if showFormBody && formBody}
				<div
					bind:this={bodyLock}
					class={[
						'relative min-h-0',
						enterComplete || exitMotion ? 'flex min-h-0 grow flex-col' : ''
					]}
					style:max-height={bodyMaxPx != null ? `${bodyMaxPx}px` : undefined}
					data-testid="in-card-body-lock"
					data-enter-complete={enterComplete}
					data-exiting={Boolean(exitMotion)}
				>
					{#if showDisplayLayer}
						<div
							bind:this={displayRoot}
							class={enterComplete || exitMotion
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
						data-testid="in-card-form-host"
						inert={!enterComplete || Boolean(exitMotion)}
					>
						{#if onSave && onDismiss}
							<InCardFormShell expanded={editing} {canSave} {onSave} {onDismiss}>
								{#snippet form()}
									{@render formBody()}
								{/snippet}
							</InCardFormShell>
						{:else}
							{@render formBody()}
						{/if}
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
</div>

{#snippet headerChrome()}
	{#if showFormBody}
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="flex items-center gap-1 pr-1"
			onclick={(e) => e.stopPropagation()}
			onpointerdown={(e) => e.stopPropagation()}
		>
			{#if docsUrl}
				<IconButton
					variant="ghost"
					href={docsUrl}
					target="_blank"
					rel="noopener noreferrer"
					icon={HelpCircle}
					size="xs"
					tooltip={m.Documentation()}
				/>
			{/if}
			{#if onDismiss}
				<IconButton
					variant="ghost"
					icon={XIcon}
					size="xs"
					tooltip={m.Close()}
					onclick={onDismiss}
					data-testid="in-card-form-dismiss"
				/>
			{/if}
		</div>
	{:else if topRight && actionsDisabled !== undefined}
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class={[
				'flex items-center gap-1 pr-1 transition-opacity',
				faded
					? 'opacity-100'
					: actionsDisabled
						? 'opacity-30'
						: 'opacity-30 group-hover:opacity-100'
			]}
			onclick={(e) => e.stopPropagation()}
			onpointerdown={(e) => e.stopPropagation()}
		>
			{@render topRight()}
		</div>
	{:else}
		{@render topRight?.()}
	{/if}
{/snippet}

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
