<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form } from '@/forms';
	import { Field, FileField } from '@/forms/fields';
	import { pb } from '@/pocketbase/index.js';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import Table, {
		ConformanceCheckSchema,
		type ConformanceCheck
	} from './wallet-form-checks-table.svelte';
	import TextareaField from '@/forms/fields/textareaField.svelte';
	import type { WalletsResponse } from '@/pocketbase/types';
	import { zodFileSchema } from '@/utils/files';

	//

	type Props = {
		onSuccess?: (wallet: WalletsResponse) => void;
		initialData?: Partial<WalletsResponse>;
		walletId?: string;
		ownerId?: string;
	};

	let { onSuccess, initialData = {}, walletId, ownerId }: Props = $props();

	//

	/**
	 * NOTE
	 * File fields conflict with the JSON field type.
	 * Cannot have nested json fields and file fields in the same form.
	 * Handle this somehow (Maybe by treating the JSON field as a string and parsing it on submit?)
	 */

	const mimeTypes = ['image/png', 'image/jpeg'];

	// TODO – Maybe use createCollectionZodSchema
	const schema = z.object({
		name: z.string().min(1, 'Name is required'),
		description: z.string(),
		playstore_url: z.string().url('Invalid URL'),
		appstore_url: z.string().url('Invalid URL'),
		logo: zodFileSchema({ mimeTypes }),
		repository: z.string().url('Invalid URL'),
		home_url: z.string().url('Invalid URL'),
		conformance_checks: z
			.array(ConformanceCheckSchema)
			.min(1, 'At least one conformance check is required')
	});

	const form = createForm<z.infer<typeof schema>>({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			if (walletId) {
				const wallet = await pb.collection('wallets').update(walletId, form.data);
				onSuccess?.(wallet);
			} else {
				const wallet = await pb.collection('wallets').create(form.data);
				onSuccess?.(wallet);
			}
		},
		options: {
			dataType: 'form'
		},
		initialData: {
			...initialData,
			conformance_checks: (initialData.conformance_checks ?? []) as ConformanceCheck[],
			logo: initialData.logo
				? new File([], initialData.logo as string, { type: mimeTypes[0] })
				: undefined
		}
	});
</script>

<Form {form} enctype="multipart/form-data" class="!space-y-8" hideRequiredIndicator>
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
	<Table {form} name="conformance_checks" options={{ label: 'Conformance Checks' }} />
</Form>
