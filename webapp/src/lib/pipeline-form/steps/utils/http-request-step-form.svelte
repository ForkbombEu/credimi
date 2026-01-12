<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
	import { Textarea } from '@/components/ui/textarea';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { HttpRequestStepForm } from './http-request-step-form.svelte.js';

	//

	let { self: form }: SelfProp<HttpRequestStepForm> = $props();

	const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];
</script>

<div class="space-y-4 p-4">
	<div class="space-y-2">
		<Label for="method">
			<T>{m.Method()}</T>
			<span class="text-red-500">*</span>
		</Label>
		<Select
			selected={{ value: form.data.method, label: form.data.method }}
			onSelectedChange={(v) => {
				if (v) form.data.method = v.value;
			}}
		>
			<SelectTrigger id="method">
				<SelectValue />
			</SelectTrigger>
			<SelectContent>
				{#each methods as method}
					<SelectItem value={method}>{method}</SelectItem>
				{/each}
			</SelectContent>
		</Select>
	</div>

	<div class="space-y-2">
		<Label for="url">
			<T>{m.URL()}</T>
			<span class="text-red-500">*</span>
		</Label>
		<Input
			id="url"
			type="url"
			placeholder="https://api.example.com/endpoint"
			bind:value={form.data.url}
		/>
	</div>

	<div class="space-y-2">
		<Label for="body">
			<T>{m.Body()}</T>
			<span class="text-muted-foreground text-sm">({m.Optional()})</span>
		</Label>
		<Textarea
			id="body"
			rows={5}
			placeholder='{"key": "value"}'
			bind:value={form.data.body}
		/>
	</div>

	<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
		<T>{m.Add_step()}</T>
	</Button>
</div>
