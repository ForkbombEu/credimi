<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { enhance } from '$app/forms';

	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { m } from '@/i18n';

	let { form } = $props();
</script>

{#if !form}
	<form method="post" use:enhance class="space-y-8">
		<div class="space-y-1">
			<T tag="h4">{m.Reset_password()}</T>
			<T>{m.Please_enter_here_a_new_password_()}</T>
		</div>
		<Label class="space-y-2">
			<span>{m.New_password()}</span>
			<Input
				type="password"
				name="password"
				id="password"
				placeholder="•••••••••••"
				required
			/>
		</Label>
		<Label class="space-y-2">
			<span>{m.New_password()}</span>
			<Input
				type="password"
				name="passwordConfirmation"
				id="passwordConfirmation"
				placeholder="•••••••••••"
				required
			/>
		</Label>
		<Button type="submit" class="w-full">{m.Reset_password()}</Button>
	</form>
{:else if form.success}
	<div class="space-y-4">
		<T tag="h4">{m.Password_reset_successfully()}</T>
		<T>{m.Please_click_the_link_in_the_email_to_reset_your_password_()}</T>
	</div>
{/if}
