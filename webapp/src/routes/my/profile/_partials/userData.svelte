<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Button from '@/components/ui-custom/button.svelte';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import { Pencil, X } from 'lucide-svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { m } from '@/i18n';
	import Separator from '@/components/ui/separator/separator.svelte';
	import T from '@/components/ui-custom/t.svelte';

	import { Form, createForm } from '@/forms';
	import { Field, FileField, CheckboxField, SelectField } from '@/forms/fields';

	import { currentUser, pb } from '@/pocketbase';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import { zod } from 'sveltekit-superforms/adapters';
	import z from 'zod';
	import { createToggleStore } from '@/components/ui-custom/utils';

	//

	const showForm = createToggleStore();

	const timezones = Intl.supportedValuesOf("timeZone") as readonly string[];

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
				$currentUser = await pb.collection('users').update($currentUser?.id!, dataToUpdate);
				showForm.off();
			},
			initialData: {
				name: $currentUser?.name,
				email: $currentUser?.email,
				emailVisibility: $currentUser?.emailVisibility,
				Timezone: $currentUser?.Timezone || 'Europe/Amsterdam',
			},
			options: {
				dataType: 'form'
			}
		})
	);
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
					({$currentUser?.emailVisibility ? 'public' : 'not public'})
				</span>
			</T>
		</div>
	</div>

	<div class="flex items-center justify-end gap-4">
		{#if $showForm}
			<Separator />
		{:else}
			<Button variant="outline" onclick={showForm.on}>
				<Icon src={Pencil} mr />
				{m.Edit_profile()}
			</Button>
		{/if}
	</div>

	{#if $showForm}
		<Form {form}>
			<Field {form} name="name" options={{ label: m.Username() }} />

			<div class="space-y-2">
				<Field {form} name="email" options={{ type: m.email() }} />

				<CheckboxField
					{form}
					name="emailVisibility"
					options={{ label: m.Show_email_to_other_users() }}
				/>
				<SelectField
					{form}
					name="Timezone"
					options={{
						label: m.Select_your_timezone(),
						items: timezones.map((tz) => ({
							value: tz,
							label: tz.replace(/_/g, ' ')
						})),
					}}
					/>
					
					
			</div>

			<FileField {form} name="avatar" />

			{#snippet submitButton({ SubmitButton })}
				<div class="flex items-center justify-end gap-2">
					<Button variant="outline" onclick={showForm.off}
						><Icon src={X} mr />{m.Cancel()}</Button
					>
					<SubmitButton><Icon src={Pencil} mr />{m.Update_profile()}</SubmitButton>
				</div>
			{/snippet}
		</Form>
	{/if}
</div>
