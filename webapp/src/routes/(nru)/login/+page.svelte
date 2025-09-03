<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { zod } from 'sveltekit-superforms/adapters';
	import z from 'zod';

	import { Form, createForm } from '@/forms';
	import { Field } from '@/forms/fields';
	import { m } from '@/i18n';
	import { goto } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { currentEmail } from './+layout.svelte';

	//

	const schema = z.object({
		email: z.string().email(),
		password: z.string()
	});

	const form = createForm({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			const { data } = form;
			const u = pb.collection('users');
			await u.authWithPassword(data.email, data.password);
			await goto('/my');
		},
		initialData: { email: currentEmail.value },
		options: { taintedMessage: null }
	});

	const { form: formData } = form;

	$effect(() => {
		currentEmail.value = $formData.email;
	});
</script>

<Form {form}>
	<Field
		{form}
		name="email"
		options={{
			type: 'email',
			label: m.Your_email(),
			placeholder: 'name@foundation.org'
		}}
	/>

	<Field
		{form}
		name="password"
		options={{
			type: 'password',
			label: m.Your_password(),
			placeholder: '•••••'
		}}
	/>

	{#snippet submitButtonContent()}
		{m.Log_in()}
	{/snippet}
</Form>
