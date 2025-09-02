<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { zod } from 'sveltekit-superforms/adapters';
	import z from 'zod';

	import Oauth from '@/auth/oauth/oauth.svelte';
	import { WelcomeSession, WelcomeBanner } from '@/auth/welcome';
	import A from '@/components/ui-custom/a.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { featureFlags } from '@/features';
	import { Form, createForm } from '@/forms';
	import { Field, CheckboxField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { goto } from '@/i18n';
	import { OrganizationInviteSession } from '@/organizations/invites';
	import { pb } from '@/pocketbase';

	//

	const schema = z
		.object({
			name: z.string(),
			email: z.string().email(),
			password: z.string().min(8),
			passwordConfirm: z.string().min(8),
			acceptTerms: z.boolean()
		})
		.refine((data) => data.password === data.passwordConfirm, m.PASSWORDS_DO_NOT_MATCH());

	const form = createForm({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			const { data } = form;
			const u = pb.collection('users');
			await u.create(data);
			await u.authWithPassword(data.email, data.password);
			await u.requestVerification(data.email);
			WelcomeSession.start();
			await goto('/my');
		}
	});

	const { form: formData } = form;

	if ($featureFlags.ORGANIZATIONS) {
		const inviteSession = OrganizationInviteSession.getData();
		if (inviteSession) $formData.email = inviteSession.email;
	}

	function getOrganization(id: string) {
		return pb.collection('organizations').getOne(id, { requestKey: null });
	}

	let disableEmail = $derived(
		$featureFlags.ORGANIZATIONS && OrganizationInviteSession.isActive()
	);
</script>

{#if $featureFlags.ORGANIZATIONS}
	{@const inviteSession = OrganizationInviteSession.getData()}
	{#if inviteSession}
		{#await getOrganization(inviteSession.organizationId) then organization}
			<WelcomeBanner class="mb-6">
				<div>
					<T>
						{@html m.you_have_been_invited_by_organization_to_join_the_platform({
							organizationName: organization.name
						})}
					</T>
					<T>{m.Please_register_using_the_provided_email_account_()}</T>
				</div>
			</WelcomeBanner>
		{/await}
	{/if}
{/if}

<T tag="h4">{m.Create_an_account()}</T>

{#if $featureFlags.OAUTH}
	<Oauth></Oauth>
{/if}

<Form {form} hideRequiredIndicator>
	<Field
		{form}
		name="email"
		options={{
			type: 'email',
			label: m.Your_email(),
			placeholder: m.namefoundation_org(),
			disabled: disableEmail
		}}
	/>

	<Field
		{form}
		name="name"
		options={{
			type: 'text',
			label: m.Full_name(),
			placeholder: m.John_Doe(),
			description: m.Organizations_and_other_users_will_identify_you_by_your_name_()
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

	<Field
		{form}
		name="passwordConfirm"
		options={{
			type: 'password',
			label: m.Confirm_password(),
			placeholder: '•••••'
		}}
	/>

	<CheckboxField {form} name="acceptTerms">
		<span class="leading-snug">
			{m.I_accept_the()}
			<A href="https://docs.credimi.io/Legal/terms-and-conditions.html" target="_blank">
				{m.Terms_and_Conditions()}
			</A>
			{m.and()}
			<A href="https://docs.credimi.io/Legal/privacy-policy.html" target="_blank">
				{m.privacy_policy()}
			</A>
		</span>
	</CheckboxField>

	{#snippet submitButtonContent()}
		{m.Create_an_account()}
	{/snippet}
</Form>

<div class="flex flex-col gap-4">
	<Separator />
	<T class="text-center text-gray-500 dark:text-gray-400" tag="small">
		{m.Already_have_an_account()}
		<A href="/login">{m.Login_here()}</A>
	</T>
</div>
