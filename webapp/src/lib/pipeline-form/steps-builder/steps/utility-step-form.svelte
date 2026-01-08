<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n/index.js';
	import Button from '@/components/ui-custom/button.svelte';
	import Input from '@/components/ui/input/input.svelte';
	import Label from '@/components/ui/label/label.svelte';
	import Textarea from '@/components/ui/textarea/textarea.svelte';
	import * as Select from '@/components/ui/select/index.js';

	import { getStepDisplayData } from '../utils/display-data.js';
	import WithLabel from '../utils/with-label.svelte';
	import type { UtilityStepForm } from './utility-step-form.svelte.js';

	//

	type Props = {
		form: UtilityStepForm;
	};

	let { form }: Props = $props();

	const { label } = $derived(getStepDisplayData(form.stepType));

	const httpMethods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];
</script>

<WithLabel {label} class="grow overflow-hidden p-4">
	{#if form.isEmail}
		<div class="flex grow flex-col gap-4 overflow-auto">
			<div class="flex flex-col gap-2">
				<Label for="recipient">{m.Recipient()}</Label>
				<Input
					id="recipient"
					type="email"
					placeholder="user@example.com"
					bind:value={form.emailData.recipient}
					required
				/>
			</div>

			<div class="flex flex-col gap-2">
				<Label for="subject">{m.Subject()}</Label>
				<Input
					id="subject"
					type="text"
					placeholder="Email subject"
					bind:value={form.emailData.subject}
				/>
			</div>

			<div class="flex flex-col gap-2">
				<Label for="body">{m.Body()}</Label>
				<Textarea
					id="body"
					placeholder="Email body"
					bind:value={form.emailData.body}
					class="min-h-32"
				/>
			</div>
		</div>
	{:else if form.isHttpRequest}
		<div class="flex grow flex-col gap-4 overflow-auto">
			<div class="flex flex-col gap-2">
				<Label for="method">{m.HTTP_Method()}</Label>
				<Select.Root
					selected={{ value: form.httpRequestData.method, label: form.httpRequestData.method }}
					onSelectedChange={(v) => {
						if (v) form.httpRequestData.method = v.value;
					}}
				>
					<Select.Trigger id="method">
						<Select.Value placeholder="Select method" />
					</Select.Trigger>
					<Select.Content>
						{#each httpMethods as method}
							<Select.Item value={method}>{method}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>

			<div class="flex flex-col gap-2">
				<Label for="url">{m.URL()}</Label>
				<Input
					id="url"
					type="url"
					placeholder="https://example.com/api"
					bind:value={form.httpRequestData.url}
					required
				/>
			</div>

			<div class="flex flex-col gap-2">
				<Label for="request-body">{m.Request_Body()}</Label>
				<Textarea
					id="request-body"
					placeholder='{"key": "value"}'
					bind:value={form.httpRequestData.body}
					class="min-h-24"
				/>
			</div>
		</div>
	{:else if form.isDebug}
		<div class="flex grow items-center justify-center p-4">
			<p class="text-muted-foreground text-center">
				Debug step will show the current workflow state during execution.
			</p>
		</div>
	{/if}
</WithLabel>

<div class="border-t p-4">
	<Button onclick={() => form.submit()} disabled={!form.canSubmit()} class="w-full">
		{m.Add_step()}
	</Button>
</div>
