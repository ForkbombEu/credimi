<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pencil } from '@lucide/svelte';
	import {
		disablePipelineNotifications,
		enablePipelineNotifications,
		getNotificationState,
		type NotificationState
	} from '$lib/pipeline/web-push';
	import { toast } from 'svelte-sonner';
	import { zod } from 'sveltekit-superforms/adapters';
	import z from 'zod/v3';

	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { Switch } from '@/components/ui/switch';
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

	let form = createForm({
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
	});

	setDashboardNavbar({
		title: m.Profile()
	});

	let pushState: NotificationState = $state('off');
	let pushUpdating = $state(false);

	$effect(() => {
		getNotificationState().then((state) => (pushState = state));
	});

	async function togglePipelineNotifications(checked: boolean) {
		pushUpdating = true;
		const result = checked
			? await enablePipelineNotifications()
			: await disablePipelineNotifications();
		pushState = await getNotificationState();
		pushUpdating = false;
		if (result.isOk) {
			toast.success(
				checked ? m.Pipeline_notifications_enabled() : m.Pipeline_notifications_disabled()
			);
		} else {
			toast.error(result.error);
		}
	}
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
					({$currentUser?.emailVisibility ? m.Public() : m.not_public()})
				</span>
			</T>
			<T tag="p">
				<span class="text-sm text-gray-400 italic">
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

	<Separator />

	<div class="flex items-center justify-between gap-4">
		<div class="space-y-1">
			<T tag="h4">{m.Pipeline_notifications()}</T>
			<T tag="p" class="text-sm text-gray-400">{m.Pipeline_notifications_description()}</T>
			{#if pushState === 'denied'}
				<T tag="p" class="text-sm text-gray-400">{m.Notifications_blocked_hint()}</T>
			{:else if pushState === 'unsupported'}
				<T tag="p" class="text-sm text-gray-400">{m.Push_notifications_unsupported()}</T>
			{/if}
		</div>
		<Switch
			aria-label={m.Pipeline_notifications()}
			checked={pushState === 'on'}
			disabled={pushState === 'unsupported' || pushUpdating}
			onCheckedChange={togglePipelineNotifications}
		/>
	</div>
</div>
