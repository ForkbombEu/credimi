<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pencil } from 'lucide-svelte';
	import { zod } from 'sveltekit-superforms/adapters';
	import z from 'zod';

	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { Form, createForm } from '@/forms';
	import { CheckboxField, Field, FileField, SelectField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { currentUser, pb } from '@/pocketbase';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';

	import { setDashboardNavbar } from '../+layout@.svelte';

	//

	const detectedTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
	const timezones = Intl.supportedValuesOf('timeZone') as readonly string[];

	const schema = createCollectionZodSchema('users').extend({
		email: z.string().email(),
		emailVisibility: z.boolean(),
		Timezone: z.string().refine((val) => timezones.includes(val), {
			message: 'Invalid timezone'
		})
	});

	let form = $derived(
		createForm({
			adapter: zod(schema),
			onSubmit: async ({ form }) => {
				const dataToUpdate = { ...form.data };
				delete dataToUpdate.verified;
				// eslint-disable-next-line @typescript-eslint/no-non-null-asserted-optional-chain
				$currentUser = await pb.collection('users').update($currentUser?.id!, dataToUpdate);
			},
			initialData: {
				name: $currentUser?.name,
				email: $currentUser?.email,
				emailVisibility: $currentUser?.emailVisibility,
				Timezone: $currentUser?.Timezone || detectedTimezone
			},
			options: {
				dataType: 'form'
			}
		})
	);

	setDashboardNavbar({
		title: m.Profile()
	});
</script>

<div class="space-y-6">
	<div class="flex flex-row items-center gap-6">
		{#if $currentUser}
			<UserAvatar class="size-20" user={$currentUser} />
		{/if}
		<div class="flex flex-col">
			<T tag="h4">{$currentUser?.name}</T>
			<T tag="p">
				{$currentUser?.email}
				<span class="ml-1 text-sm text-gray-400">
					({$currentUser?.emailVisibility ? m.public() : m.not_public()})
				</span>
			</T>
			<T tag="p">
				<span class="text-sm italic text-gray-400">
					{m.Timezone()}: {$currentUser?.Timezone || detectedTimezone}
				</span>
			</T>
		</div>
	</div>

	<Separator />

	{#key form}
		<Form {form}>
			<Field {form} name="name" options={{ label: m.Username() }} />
			<div class="space-y-2">
				<Field {form} name="email" options={{ type: m.email(), readonly: true }} />
				<CheckboxField
					{form}
					name="emailVisibility"
					options={{ label: m.Show_email_to_other_users() }}
				/>
			</div>
			<SelectField
				{form}
				name="Timezone"
				options={{
					label: m.Select_your_timezone(),
					items: timezones.map((tz) => ({
						value: tz,
						label: tz.replace(/_/g, ' ')
					}))
				}}
			/>
			<FileField {form} name="avatar" />
			{#snippet submitButton({ SubmitButton })}
				<SubmitButton><Icon src={Pencil} mr />{m.Update_profile()}</SubmitButton>
			{/snippet}
		</Form>
	{/key}
</div>
