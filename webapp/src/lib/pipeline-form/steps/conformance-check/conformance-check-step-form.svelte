<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { TriangleAlert } from '@lucide/svelte';
	import WalletActionTags from '$lib/components/wallet-action-tags.svelte';
	import * as Wallet from '$lib/wallet';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import type { ConformanceCheckStepForm } from './conformance-check-step-form.svelte.js';

	import EmptyState from '../_partials/empty-state.svelte';
	import ItemCard from '../_partials/item-card.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	//

	let { self: form }: SelfProp<ConformanceCheckStepForm> = $props();

	const hasSelection = $derived(
		form.data.standard || form.data.version || form.data.suite || form.data.test
	);

	const selectLabel = $derived.by(() => {
		if (form.state === 'select-standard') {
			return m.Standard();
		} else if (form.state === 'select-version') {
			return m.Version();
		} else if (form.state === 'select-suite') {
			return m.Suite();
		} else if (form.state === 'select-test') {
			return m.Test();
		} else if (form.state === 'select-wallet-action') {
			return m.Wallet_action();
		} else {
			return '';
		}
	});
</script>

<div>
	{#if form.state === 'loading'}
		<EmptyState>
			<Spinner />
			<T>{m.Loading()}</T>
		</EmptyState>
	{:else if form.state === 'error'}
		<EmptyState>
			<TriangleAlert size={16} />
			<T>{form.standardsWithTestSuites.error?.message}</T>
		</EmptyState>
	{:else if form.standardsWithTestSuites.current}
		{@const standards = form.standardsWithTestSuites.current}

		{#if hasSelection}
			<div class="space-y-2 border-b p-4">
				{#if form.data.standard}
					<WithLabel label={m.Standard()}>
						<ItemCard
							title={form.data.standard.name}
							onDiscard={() => form.discardStandard()}
						/>
					</WithLabel>
				{/if}
				{#if form.data.version}
					<WithLabel label={m.Version()}>
						<ItemCard
							title={form.data.version.name}
							onDiscard={() => form.discardVersion()}
						/>
					</WithLabel>
				{/if}
				{#if form.data.suite}
					<WithLabel label={m.Suite()}>
						<ItemCard
							title={form.data.suite.name}
							onDiscard={() => form.discardSuite()}
						/>
					</WithLabel>
				{/if}
				{#if form.data.test}
					<WithLabel label={m.Test()}>
						<ItemCard
							title={form.data.test.split('/').at(-1) ?? form.data.test}
							onDiscard={() => form.discardTest()}
						/>
					</WithLabel>
				{/if}
				{#if form.data.action_id && form.selectedWalletAction}
					<WithLabel label={m.Wallet_action()}>
						<ItemCard
							title={form.selectedWalletAction.name}
							onDiscard={() => form.discardWalletAction()}
						/>
					</WithLabel>
				{/if}
			</div>
		{/if}

		{#if form.state !== 'ready'}
			<WithLabel label={m.Select_item({ item: selectLabel.toLowerCase() })} class="p-4">
				<div class="space-y-2 pt-1">
					{#if form.state == 'select-standard'}
						{#each standards as standard (standard.uid)}
							<ItemCard
								title={standard.name}
								onClick={() => form.selectStandard(standard)}
							/>
						{/each}
					{:else if form.state === 'select-version'}
						{#each form.availableVersions as version (version.uid)}
							<ItemCard
								title={version.name}
								onClick={() => form.selectVersion(version)}
							/>
						{/each}
					{:else if form.state === 'select-suite'}
						{#each form.availableSuites as suite (suite.uid)}
							<ItemCard title={suite.name} onClick={() => form.selectSuite(suite)} />
						{/each}
					{:else if form.state === 'select-test'}
						{#if form.testPickerNotice.kind === 'loading'}
							<div class="flex items-center gap-2 text-sm text-muted-foreground">
								<Spinner />
								<T>{m.Loading()}</T>
							</div>
						{:else if form.testPickerNotice.kind === 'alert'}
							<div
								class="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800"
								role="alert"
							>
								<TriangleAlert size={16} class="mt-0.5 shrink-0" />
								<T>{form.testPickerNotice.message}</T>
							</div>
						{/if}
						{#each form.testOptions as option (option.test)}
							<ItemCard
								title={option.testName}
								disabled={!option.enabled}
								onClick={option.enabled ? () => form.selectTest(option) : undefined}
							/>
						{/each}
					{:else if form.state === 'select-wallet-action'}
						{#each form.genericCredentialActions as action (action.id)}
							<ItemCard
								title={action.name}
								onClick={() => form.selectWalletAction(action)}
							>
								{#snippet beforeContent()}
									{@const category = Wallet.Action.getCategoryLabel(action)}
									{#if category}
										<T class="text-xs text-muted-foreground">{category}</T>
									{/if}
								{/snippet}
								{#snippet afterContent()}
									<WalletActionTags
										{action}
										variant="secondary"
										containerClass="pt-2"
									>
										{#if !action.published}
											<Badge variant="outline">
												{m.private()}
											</Badge>
										{/if}
									</WalletActionTags>
								{/snippet}
							</ItemCard>
						{/each}
					{/if}
				</div>
			</WithLabel>
		{/if}
	{/if}
	<!-- {#if form.state === 'error'}
		<SmallErrorDisplay error={form.standardsWithTestSuites.error} />
	{/if} -->
	<!-- {#if form.state === 'select-standard'}
		<pre>{JSON.stringify(form.standardsWithTestSuites.current, null, 2)}</pre>
	{#if form.standardsWithTestSuites.current}
		<pre>{JSON.stringify(form.standardsWithTestSuites.current, null, 2)}</pre>
	{/if} -->
</div>
