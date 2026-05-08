<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import Select from '@/components/ui-custom/select.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { m } from '@/i18n';

	import type { HttpRequestStepForm } from './http-request-step-form.svelte.js';

	import WithLabel from '../_partials/with-label.svelte';
	import PlaceholderButtons from './placeholders/placeholder-buttons.svelte';

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

	{#if form.data.method === 'POST' || form.data.method === 'PUT' || form.data.method === 'PATCH'}
		<WithLabel label={m.Body()} optional>
			<CodeEditor lang="json" bind:value={form.data.body} />
		</WithLabel>
	{/if}

	<PlaceholderButtons />

	<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
		<T>{m.Add_step()}</T>
	</Button>
</div>
