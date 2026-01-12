<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Textarea } from '@/components/ui/textarea';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { EmailStepForm } from './email-step-form.svelte.js';

	//

	let { self: form }: SelfProp<EmailStepForm> = $props();
</script>

<div class="space-y-4 p-4">
	<div class="space-y-2">
		<Label for="recipient">
			<T>{m.Recipient()}</T>
			<span class="text-red-500">*</span>
		</Label>
		<Input
			id="recipient"
			type="email"
			placeholder="recipient@example.com"
			bind:value={form.data.recipient}
		/>
	</div>

	<div class="space-y-2">
		<Label for="subject">
			<T>{m.Subject()}</T>
		</Label>
		<Input id="subject" type="text" bind:value={form.data.subject} />
	</div>

	<div class="space-y-2">
		<Label for="body">
			<T>{m.Body()}</T>
		</Label>
		<Textarea id="body" rows={5} bind:value={form.data.body} />
	</div>

	<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
		<T>{m.Add_step()}</T>
	</Button>
</div>
