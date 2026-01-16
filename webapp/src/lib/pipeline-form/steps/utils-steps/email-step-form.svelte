<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Textarea } from '@/components/ui/textarea';
	import { m } from '@/i18n';

	import type { EmailStepForm } from './email-step-form.svelte.js';

	import WithLabel from '../_partials/with-label.svelte';

	//

	let { self: form }: SelfProp<EmailStepForm> = $props();
</script>

<div class="space-y-6 p-4">
	<WithLabel label={m.Recipient()} required>
		<Input
			id="recipient"
			type="email"
			placeholder="recipient@example.com"
			bind:value={form.data.recipient}
		/>
	</WithLabel>

	<WithLabel label={m.Subject()} required>
		<Input id="subject" type="text" bind:value={form.data.subject} />
	</WithLabel>

	<WithLabel label={m.Body()} optional>
		<Textarea id="body" rows={5} bind:value={form.data.body} />
	</WithLabel>

	<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
		<T>{m.Add_step()}</T>
	</Button>
</div>
