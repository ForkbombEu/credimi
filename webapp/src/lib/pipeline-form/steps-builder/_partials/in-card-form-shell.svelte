<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { untrack } from 'svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n/index.js';

	//

	type Props = {
		expanded: boolean;
		canSave: boolean;
		onSave: () => void;
		onDismiss: () => void;
		/** Scrollable form fields inside the card body. */
		form: Snippet;
	};

	let { expanded, canSave, onSave, onDismiss, form }: Props = $props();

	let mounted = $state(untrack(() => expanded));

	// Mount as soon as edit opens. Enter/exit height motion is owned by InCardHost
	// (shrink targets the summary card height, not zero).
	$effect(() => {
		if (!expanded) return;
		if (untrack(() => mounted)) return;
		untrack(() => {
			mounted = true;
		});
	});

	// Stay mounted while the parent holds the form for the exit animation.
	// Parent clears the held mode after playInCardExit completes.

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
