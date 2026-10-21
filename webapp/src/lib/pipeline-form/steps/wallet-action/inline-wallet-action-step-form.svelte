<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { WithLabel } from '$pipeline-form/steps/_partials/index.js';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { InlineWalletActionStepForm } from './inline-wallet-action-step-form.svelte.js';

	import { getDeviceLabel, getVersionLabel } from './labels.js';

	let { self: form }: SelfProp<InlineWalletActionStepForm> = $props();
</script>

{#if form.data}
	<div class="flex flex-col gap-4 p-4">
		<T class="text-sm text-muted-foreground">
			{form.data.kind === 'actions' ? m.Actions_edit_notice() : m.Inline_action_edit_notice()}
		</T>
		{#if form.data.kind === 'actions'}
			<WithLabel label={m.Wallet_actions()}>
				<ol class="list-decimal pl-4 font-mono text-xs text-muted-foreground">
					{#each form.data.actions as action, index (index)}
						<li>{'id' in action ? action.id : m.Custom_code()}</li>
					{/each}
				</ol>
			</WithLabel>
		{/if}
		<WithLabel label={m.Version()}>
			<p class="text-sm">{getVersionLabel(form.data.version)}</p>
		</WithLabel>
		<WithLabel label={m.Device()}>
			<p class="text-sm">{getDeviceLabel(form.data.device)}</p>
		</WithLabel>
		{#if form.data.parameters && Object.keys(form.data.parameters).length > 0}
			<WithLabel label={m.parameters()}>
				<p class="font-mono text-xs text-muted-foreground">
					{Object.keys(form.data.parameters).join(', ')}
				</p>
			</WithLabel>
		{/if}
	</div>
{/if}
