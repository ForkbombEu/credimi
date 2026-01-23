<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { zod } from 'sveltekit-superforms/adapters';
	import z from 'zod/v3';

	import A from '@/components/ui-custom/a.svelte';
	import { Form, createForm } from '@/forms';
	import { Field } from '@/forms/fields';
	import { goto, m } from '@/i18n';
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

	<div>
		<Field
			{form}
			name="password"
			options={{
				type: 'password',
				label: m.Your_password(),
				placeholder: '•••••'
			}}
		/>
		<A class="block text-right text-sm" href="/forgot-password">{m.Forgot_password()}</A>
	</div>

	{#snippet submitButton({ SubmitButton })}
		<SubmitButton class="w-full">
			{m.Log_in()}
		</SubmitButton>
	{/snippet}
</Form>
