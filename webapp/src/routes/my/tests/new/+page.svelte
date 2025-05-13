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

	//

	type FormState = 'select-tests' | 'fill-values';

	let formState = $state<FormState>('select-tests');

	//

	const tabs: { id: FormState; label: string }[] = [
		{ id: 'select-tests', label: '1. Standard and test suite' },
		{ id: 'fill-values', label: '2. Key values and JSONs' }
	];

	//

	let selectedCustomChecksIds = $state<string[]>([]);
	const selectedCustomChecks = $derived(
		data.customChecks.filter((check) => selectedCustomChecksIds.includes(check.id))
	);
</script>

<!--  -->

<div class="bg-background relative mx-auto w-full max-w-screen-xl rounded-md shadow-sm">
	<div class="space-y-12 p-8 pb-0">
		<div>
			<BackButton href="/my">Back to dashboard</BackButton>
			<T tag="h1">Compliance tests</T>
		</div>

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

	{#if formState === 'select-tests'}
		<SelectTestForm
			standards={data.standardsAndTestSuites}
			customChecks={data.customChecks}
			onSelectTests={async (data) => {
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
