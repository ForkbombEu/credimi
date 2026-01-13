<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import Select from '@/components/ui-custom/select.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Textarea } from '@/components/ui/textarea';
	import { m } from '@/i18n';

	import type { HttpRequestStepForm } from './http-request-step-form.svelte.js';

	import WithLabel from '../_partials/with-label.svelte';

	//

	let { self: form }: SelfProp<HttpRequestStepForm> = $props();

	const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];
</script>

<div class="space-y-6 p-4">
	<WithLabel label={m.Method()}>
		<Select
			items={methods.map((method) => ({ value: method, label: method }))}
			bind:value={form.data.method}
		/>
	</WithLabel>

	<WithLabel label={m.URL()}>
		<Input
			id="url"
			type="url"
			placeholder="https://api.example.com/endpoint"
			bind:value={form.data.url}
		/>
	</WithLabel>

	<WithLabel label={m.Body()} optional>
		<Textarea id="body" rows={5} placeholder={`{"key": "value"}`} bind:value={form.data.body} />
	</WithLabel>

	<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
		<T>{m.Add_step()}</T>
	</Button>
</div>
