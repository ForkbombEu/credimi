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
	import { loginUser } from '@/webauthn/index';

	import { currentEmail } from '../+layout.svelte';

	const schema = z.object({
		email: z.string().email()
	});

	const form = createForm({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			const { data } = form;
			await loginUser(data.email);
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
			id: 'email',
			type: 'email',
			label: m.Your_email(),
			placeholder: 'name@foundation.org'
		}}
	/>

	{#snippet submitButtonContent()}
		{m.Log_in_with_webauthn()}
	{/snippet}
</Form>
