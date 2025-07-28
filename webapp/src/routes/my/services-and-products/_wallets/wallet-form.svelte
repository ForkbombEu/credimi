<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form } from '@/forms';
	import { Field, FileField, CheckboxField } from '@/forms/fields';
	import { pb } from '@/pocketbase/index.js';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import Table, { ConformanceCheckSchema } from './wallet-form-checks-table.svelte';
	import TextareaField from '@/forms/fields/textareaField.svelte';
	import type { WalletsResponse } from '@/pocketbase/types';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import _ from 'lodash';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { m } from '@/i18n';
	import { InfoIcon } from 'lucide-svelte';
	import T from '@/components/ui-custom/t.svelte';

	//

	type Props = {
		onSuccess?: (wallet: WalletsResponse) => void;
		initialData?: Partial<WalletsResponse>;
		walletId?: string;
	};

	let { onSuccess, initialData = {}, walletId }: Props = $props();

	//

	/**
	 * NOTE
	 * File fields conflict with the JSON field type.
	 * Cannot have nested json fields and file fields in the same form.
	 * Handle this somehow (Maybe by treating the JSON field as a string and parsing it on submit?)
	 */

	const schema = createCollectionZodSchema('wallets')
		.omit({
			owner: true
		})
		.extend({
			conformance_checks: z.array(ConformanceCheckSchema).nullable()
		});

	const form = createForm<z.infer<typeof schema>>({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			let wallet: WalletsResponse;
			if (walletId) {
				// Temp fix
				const data = _.omit(form.data, 'conformance_checks');
				wallet = await pb.collection('wallets').update(walletId, data);
			} else {
				wallet = await pb.collection('wallets').create(form.data);
			}
			onSuccess?.(wallet);
		},
		options: {
			dataType: 'form'
		},
		initialData: _.omit(initialData, 'logo', 'conformance_checks')
		// TODO - Fix edit form for conformance_checks
		// {
		// 	..._.omit(initialData, 'logo', "conformance_checks"),
		// 	// conformance_checks: initialData.conformance_checks as NonEmptyArray<ConformanceCheck>
		// }
	});
</script>

<Form {form} enctype="multipart/form-data" class="!space-y-8">
	<div class="flex justify-end">
		<CheckboxField
			{form}
			name="published"
			options={{
				label: m.Published()
			}}
		/>
	</div>
	<Field
		{form}
		name="name"
		options={{
			type: 'text',
			label: 'App Name',
			placeholder: 'Enter app name'
		}}
	/>
	<TextareaField
		{form}
		name="description"
		options={{
			label: 'Description',
			placeholder: 'Enter app description'
		}}
	/>
	<FileField
		{form}
		name="logo"
		options={{
			label: 'Logo',
			placeholder: 'Upload logo'
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
	<!-- {walletId}
	{#if !walletId} -->
	<!-- @ts-ignore -->
	<!-- TODO - Typecheck -->
	<Table form={form as any} name="conformance_checks" options={{ label: 'Conformance Checks' }} />
	<!-- {:else}
		<Alert variant="info" icon={InfoIcon}>
			<T>Editing conformance checks for wallets is temporary disabled.</T>
		</Alert>
	{/if} -->
</Form>
