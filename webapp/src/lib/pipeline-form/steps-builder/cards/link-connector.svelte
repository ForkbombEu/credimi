<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Link2, Unlink2 } from '@lucide/svelte';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';

	//

	type Props = {
		linked: boolean;
		disabled?: boolean;
		onToggle: () => void;
	};

	let { linked, disabled = false, onToggle }: Props = $props();
</script>

<!-- Sits in the gap above a step card: linked steps are joined by a line and retried together. -->
<div
	class="pointer-events-none absolute inset-x-0 -top-3 z-10 flex h-3 items-center justify-center"
>
	{#if linked}
		<div class="absolute -inset-y-1 left-1/2 w-0.5 -translate-x-1/2 bg-primary"></div>
	{/if}
	<IconButton
		icon={linked ? Link2 : Unlink2}
		size="mini"
		variant={linked ? 'default' : 'outline'}
		class={['pointer-events-auto rounded-full', { 'text-muted-foreground': !linked }]}
		tooltip={linked ? m.Unlink_from_previous_step() : m.Link_with_previous_step()}
		aria-label={linked ? m.Unlink_from_previous_step() : m.Link_with_previous_step()}
		aria-pressed={linked}
		{disabled}
		onclick={onToggle}
	/>
</div>
