<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ConfigureChecksFormComponent } from './configure-checks-form';
	import { SelectChecksFormComponent } from './select-checks-form';
	import { StartChecksForm, type StartChecksFormProps } from './start-checks-form.svelte.js';
	import * as Tabs from '@/components/ui/tabs/index.js';

	//

	const props: StartChecksFormProps = $props();
	const form = new StartChecksForm(props);

	//

	type FormState = typeof form.state;

	const tabs: { id: FormState; label: string }[] = [
		{ id: 'select-tests', label: '1. Standard and test suite' },
		{ id: 'fill-values', label: '2. Key values and JSONs' }
	];

	$effect(() => {
		if (form.state === 'fill-values') {
			scrollTo({ top: 0, behavior: 'instant' });
		}
	});
</script>

<Tabs.Root value={form.state} class="w-full">
	<Tabs.List class="flex">
		<Tabs.Trigger
			value={tabs[0].id}
			class="data-[state=inactive]:hover:bg-primary/10 grow data-[state=inactive]:text-black"
			onclick={() => {
				form.backToSelectTests();
			}}
		>
			{tabs[0].label}
		</Tabs.Trigger>
		<Tabs.Trigger value={tabs[1].id} class="grow" disabled={form.state === 'select-tests'}>
			{tabs[1].label}
		</Tabs.Trigger>
	</Tabs.List>
</Tabs.Root>

{#if form.state === 'select-tests'}
	<SelectChecksFormComponent form={form.selectChecksForm} />
{:else if form.state === 'fill-values' && form.configureChecksFormProps}
	<ConfigureChecksFormComponent {...form.configureChecksFormProps} />
{/if}
