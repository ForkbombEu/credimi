<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module>
	export const queryParams = {
		customCheckId: 'custom_check_id'
	};
</script>

<script lang="ts">
	import TestsDataForm from './_partials/tests-data-form.svelte';
	import * as Tabs from '@/components/ui/tabs/index.js';
	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';
	import { m } from '@/i18n';
	import { page } from '$app/state';
	import SelectTestsForm from './_partials/select-tests-form/select-tests-form.svelte';
	import type { SelectTestsFormData } from './_partials/select-tests-form/select-tests-form.svelte.js';
	import { getTestsConfigsFields } from './_partials/tests-configs-form/utils';
	import type { TestsConfigsFields } from './_partials/tests-configs-form/types';

	//

	let { data } = $props();

	//

	type FormState = 'select-tests' | 'fill-values';
	let formState = $state<FormState>('select-tests');

	const tabs: { id: FormState; label: string }[] = [
		{ id: 'select-tests', label: '1. Standard and test suite' },
		{ id: 'fill-values', label: '2. Key values and JSONs' }
	];

	//

	let selectedCustomChecksIds = $state<string[]>([]);
	const selectedCustomChecks = $derived(
		data.customChecks.filter((check) => selectedCustomChecksIds.includes(check.id))
	);

	//

	let compositeTestId = $state('');
	let testsConfigsFields = $state<TestsConfigsFields>();

	async function handleSelection(data: SelectTestsFormData) {
		compositeTestId = data.standardId + '/' + data.versionId;
		selectedCustomChecksIds = data.customChecks;

		if (data.tests.length > 0) {
			testsConfigsFields = await getTestsConfigsFields(compositeTestId, data.tests);
		}

		scrollTo({ top: 0, behavior: 'instant' });
		formState = 'fill-values';
	}

	//

	const customCheckId = $derived(page.url.searchParams.get(queryParams.customCheckId));

	$effect(() => {
		if (!customCheckId) return;
		const customCheck = data.customChecks.find((check) => check.id === customCheckId);
		if (!customCheck) return;

		compositeTestId = customCheck.standard_and_version;
		selectedCustomChecksIds = [customCheckId];
		formState = 'fill-values';
	});
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
					<Tabs.Trigger
						value={tabs[1].id}
						class="grow"
						disabled={!Boolean(testsConfigsFields)}
					>
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
				onSubmit={handleSelection}
			/>
		{:else if formState === 'fill-values'}
			<TestsDataForm
				data={testsConfigsFields}
				testId={compositeTestId}
				customChecks={selectedCustomChecks}
			/>
		{/if}
	</div>
</FocusPageLayout>
