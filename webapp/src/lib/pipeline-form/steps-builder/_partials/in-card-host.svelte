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
	import { onDestroy, tick, untrack } from 'svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n/index.js';

	import {
		inCardFormHostClass,
		playInCardEnterLayout,
		playInCardExitLayout
	} from '../in-card-layout.js';
	import InCardFormShell from './in-card-form-shell.svelte';
	import { cancelMotion, type MotionHandle } from './in-card-motion.js';
	import StepCardDisplay from './step-card-display.svelte';
	import { useCardShell } from './use-card-shell.svelte.js';

	//

	type Props = {
		step: EnrichedStep;
		builder: StepsBuilder;
		editing?: boolean;
		/** Enter-ready: parent `editing && session.inCard.phase === 'still'`. */
		expandReady?: boolean;
		selected?: boolean;
		hovered?: boolean;
		/** Sibling of the card being edited in place: dimmed and non-interactive (except the pencil). */
		faded?: boolean;
		/** Max height of the card while editing, in px. Composer passes `session.cardFillMaxPx`. */
		maxHeightPx?: number;
		/** Paired with held-form clear — Twin-pane `noteExitComplete`. */
		onExitUnlock?: () => void;
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
		maxHeightPx,
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

	/** True after enter grow finishes; layout/inert only — not bound by parents. */
	let enterComplete = $state(false);

	/** Keep summary painted until enter crossfade finishes (or during exit). */
	let showDisplayLayer = $state(true);
	/** Body cap (card max − chrome); kept on the lock so the save bar does not reflow. */
	let bodyMaxPx = $state<number | undefined>(undefined);
	/** Set while exit layout handle is live — form stays inert; not session `exiting`. */
	let exitMotion = $state<MotionHandle | undefined>(undefined);

	let fadedRoot: HTMLElement | null = $state(null);
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
		const wrap = fadedRoot;
		const card = wrap?.firstElementChild instanceof HTMLElement ? wrap.firstElementChild : null;
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
						shell.completeExit();
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
						shell.completeExit();
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
	bind:this={fadedRoot}
	class={[
		'flex min-h-0 flex-col transition-opacity duration-200',
		faded && 'pointer-events-none opacity-40'
	]}
	style:max-height={showFormBody && maxHeightPx != null ? `${maxHeightPx}px` : undefined}
>
	<StepCardDisplay
		{step}
		{editing}
		{selected}
		{hovered}
		{footer}
		topRight={hostHeader}
		body={showFormBody ? overlayBody : undefined}
	/>
</div>

{#snippet hostHeader()}
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
			<IconButton
				variant="ghost"
				icon={XIcon}
				size="xs"
				tooltip={m.Close()}
				onclick={() => builder.exitFormState()}
				data-testid="in-card-form-dismiss"
			/>
		</div>
	{:else}
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class={[
				'flex items-center gap-1 pr-1 transition-opacity',
				faded ? 'opacity-100' : 'opacity-30 group-hover:opacity-100'
			]}
			onclick={(e) => e.stopPropagation()}
			onpointerdown={(e) => e.stopPropagation()}
		>
			{@render topRight()}
		</div>
	{/if}
{/snippet}

{#snippet overlayBody({ details, footer }: { details: Snippet; footer: Snippet })}
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
				{@render details()}
				{@render footer()}
			</div>
		{/if}

		<div
			bind:this={formHost}
			data-testid="in-card-form-host"
			inert={!enterComplete || Boolean(exitMotion)}
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
