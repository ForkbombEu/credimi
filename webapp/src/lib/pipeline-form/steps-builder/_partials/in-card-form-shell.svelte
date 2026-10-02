<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { onDestroy, untrack } from 'svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n/index.js';

	import { cancelMotion, collapseRegion, type MotionHandle } from './in-card-motion.js';

	//

	type Props = {
		expanded: boolean;
		canSave: boolean;
		onSave: () => void;
		onDismiss: () => void;
		/** Fired after the collapse animation finishes (or immediately if reduced motion). */
		onCollapsed?: () => void;
		/** Scrollable form fields inside the card body. */
		form: Snippet;
	};

	let { expanded, canSave, onSave, onDismiss, onCollapsed, form }: Props = $props();

	let mounted = $state(untrack(() => expanded));
	let region: HTMLElement | null = $state(null);

	let motion: MotionHandle | undefined;
	let transition = 0;

	// Mount as soon as edit opens. Enter fade/grow is owned by StepCardDisplay
	// (summary stays visible until scroll settles, then crossfades).
	$effect(() => {
		if (!expanded) return;
		if (untrack(() => mounted)) return;
		untrack(() => {
			transition++;
			mounted = true;
		});
	});

	// Collapse on dismiss / save.
	$effect(() => {
		if (expanded) return;
		if (!untrack(() => mounted)) return;
		const current = ++transition;
		const el = untrack(() => region);
		untrack(() => {
			if (el) {
				motion = collapseRegion(el, {
					onComplete: () => {
						if (current !== transition) return;
						mounted = false;
						onCollapsed?.();
					}
				});
			} else {
				mounted = false;
				onCollapsed?.();
			}
		});
	});

	onDestroy(() => {
		transition++;
		motion?.cancel();
		cancelMotion(region);
	});

	function onWindowKeydown(event: KeyboardEvent) {
		if (!expanded || event.key !== 'Escape') return;
		if (event.defaultPrevented || event.isComposing) return;
		event.preventDefault();
		onDismiss();
	}
</script>

<svelte:window onkeydown={onWindowKeydown} />

{#if mounted}
	<div
		bind:this={region}
		class="flex min-h-0 grow flex-col"
		inert={!expanded}
		data-testid="in-card-form-shell"
		data-expanded={expanded}
	>
		<div class="min-h-0 grow overflow-y-auto" data-testid="in-card-form-body">
			{@render form()}
		</div>

		<div class="shrink-0 border-t p-3">
			<Button
				class="w-full"
				disabled={!canSave}
				onclick={onSave}
				data-testid="in-card-form-save"
			>
				{m.Save()}
			</Button>
		</div>
	</div>
{/if}
