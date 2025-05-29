<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { getVariables, type FieldsResponse } from './_partials/logic';
	import TestsDataForm from './_partials/tests-data-form.svelte';
	import * as Tabs from '@/components/ui/tabs/index.js';
	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';
	import { m } from '@/i18n';
	import { page } from '$app/state';
	import SelectTestsForm from './_partials/select-tests-form/select-tests-form.svelte';

	//

	let { data } = $props();

	let d = $state<FieldsResponse>();
	let compositeTestId = $state('');

	let searchParams = $state(page.url.searchParams);
	let testId = $state(searchParams.get('test_id') || undefined);

	//

	type FormState = 'select-tests' | 'fill-values';

	let formState = $state<FormState>(testId ? 'fill-values' : 'select-tests');

	//

	const tabs: { id: FormState; label: string }[] = [
		{ id: 'select-tests', label: '1. Standard and test suite' },
		{ id: 'fill-values', label: '2. Key values and JSONs' }
	];

	//

	let selectedCustomChecksIds = $state<string[]>(testId ? [testId] : []);
	const selectedCustomChecks = $derived(
		data.customChecks.filter((check) => selectedCustomChecksIds.includes(check.id))
	);
</script>

<!--  -->

<FocusPageLayout
	title={m.Start_a_new_conformance_check()}
	description={m.Start_a_new_conformance_check_description()}
	backButton={{ href: '/my', title: m.Back_to_dashboard() }}
>
	{#snippet top()}
		<div class="space-y-12 pb-0 pt-8">
			<Tabs.Root value={formState} class="w-full">
				<Tabs.List class="bg-secondary flex">
					<Tabs.Trigger
						value={tabs[0].id}
						class="data-[state=inactive]:hover:bg-primary/10 grow data-[state=inactive]:text-black"
						onclick={() => {
							formState = 'select-tests';
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
	{/snippet}

	<div class="bg-background relative w-full rounded-md shadow-sm">
		{#if formState === 'select-tests'}
			<SelectTestsForm
				standards={data.standardsAndTestSuites}
				customChecks={data.customChecks}
				onSubmit={async (data) => {
					compositeTestId = data.standardId;
					//
					selectedCustomChecksIds = data.customChecks;
					//
					if (data.tests.length > 0) {
						d = await getVariables(data.standardId, data.tests);
					}
					//
					scrollTo({ top: 0, behavior: 'instant' });
					formState = 'fill-values';
				}}
			/>
		{:else if formState === 'fill-values'}
			<TestsDataForm data={d} testId={compositeTestId} customChecks={selectedCustomChecks} />
		{/if}
	</div>
</FocusPageLayout>
