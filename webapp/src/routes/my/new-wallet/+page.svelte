<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form } from '@/forms';
	import { Field } from '@/forms/fields';
	import { pb } from '@/pocketbase/index.js';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import Table from './conformance_checks_table.svelte';

	//

	let { data } = $props();
	const { workflows } = $derived(data);

	const schema = z.object({
		name: z.string().min(1, 'Name is required'),
		description: z.string(),
		playstore_url: z.string().url('Invalid URL'),
		appstore_url: z.string().url('Invalid URL'),
		repository: z.string().url('Invalid URL'),
		home_url: z.string().url('Invalid URL'),
		conformance_checks: z.array(z.string())
	});

	const form = createForm<z.infer<typeof schema>>({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			const { data } = form;
			pb.collection('wallets').create({
				...data,
				conformace_checks: { runs: data.conformance_checks }
			});
		},
		initialData: {
			name: '',
			description: '',
			playstore_url: '',
			appstore_url: '',
			repository: '',
			home_url: '',
			conformance_checks: []
		}
	});

	const tableData = $derived(
		workflows.map((w) => ({
			runId: w.runId,
			// @ts-ignore
			standard: atob(execution.memo.fields.standard.data),
			// @ts-ignore
			test: atob(execution.memo.fields.test.data)
		}))
	);
</script>

<Form {form}>
	<Field
		{form}
		name="name"
		options={{
			type: 'text',
			label: 'App Name',
			placeholder: 'Enter app name'
		}}
	/>
	<Field
		{form}
		name="description"
		options={{
			type: 'textarea',
			label: 'Description',
			placeholder: 'Enter app description'
		}}
	/>
	<Field
		{form}
		name="playstore_url"
		options={{
			type: 'url',
			label: 'Play Store URL',
			placeholder: 'Enter Play Store URL'
		}}
	/>
	<Field
		{form}
		name="appstore_url"
		options={{
			type: 'url',
			label: 'App Store URL',
			placeholder: 'Enter App Store URL'
		}}
	/>
	<Field
		{form}
		name="repository"
		options={{
			type: 'url',
			label: 'Repository URL',
			placeholder: 'Enter repository URL'
		}}
	/>
	<Field
		{form}
		name="home_url"
		options={{
			type: 'url',
			label: 'Home URL',
			placeholder: 'Enter home URL'
		}}
	/>
	<Table
		data={tableData}
		{form}
		name="conformance_checks"
		options={{ label: 'Conformance Checks' }}
	/>
</Form>
