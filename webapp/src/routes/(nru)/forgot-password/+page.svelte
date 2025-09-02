<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { enhance } from '$app/forms';

	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button/index.js';
	import { Input } from '@/components/ui/input/index.js';
	import Label from '@/components/ui/label/label.svelte';

	let { form } = $props();

	// TODO - Translations
</script>

{#if !form}
	<form method="post" use:enhance class="space-y-8">
		<div class="space-y-1">
			<T tag="h4">Forgot password?</T>
			<T>Please enter here your email to recover your password.</T>
		</div>
		<Label class="space-y-2">
			<span>Your email</span>
			<Input type="email" name="email" id="email" placeholder="name@foundation.org" required />
		</Label>
		<Button type="submit" class="w-full">Recover password</Button>
	</form>
{:else if form.success}
	<div class="space-y-4">
		<T tag="h4">Reset email sent successfully!</T>
		<T>Please click the link in the email to reset your password.</T>
	</div>
{/if}
