<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
	import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

	import { HelpCircle, XIcon } from '@lucide/svelte';
	import { type Comp, Render } from '$lib/renderable';
	import { onDestroy, untrack } from 'svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n/index.js';

	import type { InCardHostChrome } from './in-card-host-chrome.js';

	import { inCardFormHostClass } from '../in-card-layout.js';
	import InCardFormShell from './in-card-form-shell.svelte';
	import { createInCardLayoutMachine } from './in-card-host-machine.svelte.js';
	import StepCardDisplay from './step-card-display.svelte';
	import { useCardShell } from './use-card-shell.svelte.js';

	//

	export type { InCardHostChrome };

	type Props = InCardHostChrome & {
		step: EnrichedStep;
		builder: StepsBuilder;
		topRight: Snippet;
		footer?: Snippet | Comp<Component<any>>;
	};

	let {
		step,
		builder,
		editing = false,
		expandReady = false,
		selected = false,
		hovered = false,
		faded = false,
		cardFillMaxPx,
		onExitUnlock,
		topRight,
		footer
	}: Props = $props();

	const shell = useCardShell(
		() => builder,
		() => editing,
		() => {
			onExitUnlock?.();
		}
	);
	const showFormBody = $derived(shell.mode !== null);
	const canSave = $derived(Boolean(editing && shell.mode?.form.canSave()));
	const docsUrl = $derived(shell.mode?.config.docsUrl);

	let cardRoot: HTMLElement | null = $state(null);
	let bodyLock: HTMLElement | null = $state(null);
	let displayRoot: HTMLElement | null = $state(null);
	let formHost: HTMLElement | null = $state(null);

	const layout = createInCardLayoutMachine({
		getEls: () => {
			const card = cardRoot;
			const lock = bodyLock;
			const display = displayRoot;
			const form = formHost;
			if (!card || !lock || !display || !form) return null;
			return { card, lock, display, form };
		},
		getCardFillMaxPx: () => cardFillMaxPx,
		onDone: () => shell.completeExit()
	});

	$effect(() => {
		void showFormBody;
		void expandReady;
		void editing;
		void bodyLock;
		void displayRoot;
		void formHost;
		void cardRoot;
		untrack(() => layout.sync({ showFormBody, expandReady, editing }));
	});

	onDestroy(() => layout.destroy());

	const showDisplayLayer = $derived(layout.current !== 'settled');
	/** Sole writer of form-host classes (`inCardFormHostClass`); layout never sets className. */
	const lockSettledLayout = $derived(
		layout.current === 'settled' || layout.current === 'exiting'
	);

	function stopHeaderBubble(e: Event) {
		e.stopPropagation();
	}
</script>

<div
	class={[
		'flex min-h-0 flex-col transition-opacity duration-200',
		faded && 'pointer-events-none opacity-40'
	]}
	style:max-height={showFormBody && cardFillMaxPx != null ? `${cardFillMaxPx}px` : undefined}
>
	<StepCardDisplay
		bind:cardRoot
		{step}
		{editing}
		{selected}
		{hovered}
		{footer}
		topRight={hostHeader}
		body={showFormBody ? overlayBody : undefined}
	/>
</div>

{#snippet headerToolbar(className: string | Array<string | false | undefined>, children: Snippet)}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class={className} onclick={stopHeaderBubble} onpointerdown={stopHeaderBubble}>
		{@render children()}
	</div>
{/snippet}

{#snippet hostHeader()}
	{#if showFormBody}
		{@render headerToolbar('flex items-center gap-1 pr-1', editToolbar)}
	{:else}
		{@render headerToolbar(
			[
				'flex items-center gap-1 pr-1 transition-opacity',
				faded ? 'opacity-100' : 'opacity-30 group-hover:opacity-100'
			],
			idleToolbar
		)}
	{/if}
{/snippet}

{#snippet editToolbar()}
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
	<IconButton
		variant="ghost"
		icon={XIcon}
		size="xs"
		tooltip={m.Close()}
		onclick={() => builder.exitFormState()}
		data-testid="in-card-form-dismiss"
	/>
{/snippet}

{#snippet idleToolbar()}
	{@render topRight()}
{/snippet}

{#snippet overlayBody({ details, footer }: { details: Snippet; footer: Snippet })}
	<div
		bind:this={bodyLock}
		class={['relative min-h-0', lockSettledLayout && 'flex min-h-0 grow flex-col']}
		style:max-height={layout.bodyMaxPx != null ? `${layout.bodyMaxPx}px` : undefined}
		data-testid="in-card-body-lock"
		data-enter-complete={layout.current === 'settled'}
		data-exiting={layout.current === 'exiting'}
	>
		{#if showDisplayLayer}
			<div
				bind:this={displayRoot}
				class={lockSettledLayout
					? 'pointer-events-none absolute top-0 right-0 left-0 w-full'
					: undefined}
				data-testid="in-card-display-body"
			>
				{@render details()}
				{@render footer()}
			</div>
		{/if}

		<div
			bind:this={formHost}
			class={inCardFormHostClass(lockSettledLayout)}
			data-testid="in-card-form-host"
			inert={layout.current !== 'settled'}
		>
			<InCardFormShell
				expanded={editing}
				{canSave}
				onSave={() => {
					shell.mode?.form.commit();
				}}
				onDismiss={() => builder.exitFormState()}
			>
				{#snippet form()}
					{#if shell.mode}
						<Render item={shell.mode.form} />
					{/if}
				{/snippet}
			</InCardFormShell>
		</div>
	</div>
{/snippet}
