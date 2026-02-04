<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { TriangleAlert } from '@lucide/svelte';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
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
			</div>
		{/if}

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
					{#each form.availableTests as test (test)}
						{@const testName = test.split('/').at(-1)?.replaceAll('+', ' ')}
						<ItemCard title={testName ?? test} onClick={() => form.selectTest(test)} />
					{/each}
				{/if}
			</div>
		</WithLabel>
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
