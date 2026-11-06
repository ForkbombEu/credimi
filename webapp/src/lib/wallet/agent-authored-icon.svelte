<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { BotIcon } from '@lucide/svelte';

	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	//

	/** Shown after the name of a wallet action the action author agent wrote (`isAgentAuthored`). */
	type Props = { authored: boolean };

	let { authored }: Props = $props();
</script>

{#if authored}
	<Tooltip>
		{#snippet child({ props })}
			<span
				{...props}
				class="ml-0.5 inline-flex translate-y-px text-primary"
				aria-label={m.Agent_authored_action()}
				data-testid="agent-authored-action"
			>
				<BotIcon class="size-4" strokeWidth={1.5} />
			</span>
		{/snippet}
		{#snippet content()}
			<p>{m.Agent_authored_action()}</p>
		{/snippet}
	</Tooltip>
{/if}
