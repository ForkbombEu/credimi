<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import SelectTestForm from './_partials/select-test-form.svelte';
	import { getVariables, type FieldsResponse } from './_partials/logic';
	import TestsDataForm from './_partials/tests-data-form.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Tabs from '@/components/ui/tabs/index.js';
	import BackButton from '$lib/layout/back-button.svelte';

	//

	let { data } = $props();

	let d = $state<FieldsResponse>();
	let compositeTestId = $state('');

	const tabs = [
		{ id: 'standard', label: '1. Standard and test suite' },
		{ id: 'values', label: '2. Key values and JSONs' }
	] as const;

	type Tab = (typeof tabs)[number]['id'];

	const currentTab = $derived<Tab>(d ? 'values' : 'standard');
</script>

<!--  -->

<div class="relative mx-auto w-full max-w-screen-xl rounded-md bg-background shadow-sm">
	<div class="space-y-12 p-8 pb-0">
		<div>
			<BackButton href="/my">Back to dashboard</BackButton>
			<T tag="h1">Compliance tests</T>
		</div>

		<Tabs.Root value={currentTab} class="w-full">
			<Tabs.List class="flex bg-secondary">
				<Tabs.Trigger
					value={tabs[0].id}
					class="grow data-[state=inactive]:text-black data-[state=inactive]:hover:bg-primary/10"
					onclick={() => {
						d = undefined;
					}}
				>
					{tabs[0].label}
				</Tabs.Trigger>
				<Tabs.Trigger value={tabs[1].id} class="grow" disabled={!Boolean(d)}>
					{tabs[1].label}
				</Tabs.Trigger>
			</Tabs.List>
		</Tabs.Root>
	</div>

	{#if !d}
		<SelectTestForm
			standards={data.standardsAndTestSuites}
			onSelectTests={(data) => {
				compositeTestId = data.standardId;
				getVariables(data.standardId, data.tests).then((res) => {
					d = res;
					scrollTo({ top: 0, behavior: 'instant' });
				});
			}}
		/>
	{:else}
		<TestsDataForm data={d} testId={compositeTestId} />
	{/if}
</div>
