<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { TestConfigForm } from './test-config-form.svelte.js';
	import { TestConfigJsonFormComponent } from '$lib/start-checks-form/test-config-json-form';
	import { TestConfigFieldsFormComponent } from '$lib/start-checks-form/test-config-fields-form';
	import Alert from '@/components/ui-custom/alert.svelte';
	import { Info } from 'lucide-svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { Label } from '@/components/ui/label';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';

	type Props = {
		form: TestConfigForm;
	};

	const { form }: Props = $props();
</script>

<div class="flex flex-col gap-8 md:flex-row">
	<div class="min-w-0 shrink-0 grow basis-1">
		<div class="mb-8 space-y-2">
			<Label>{m.Fields()}</Label>
			<Separator />
		</div>

		{#if form.mode == 'fields'}
			<TestConfigFieldsFormComponent form={form.fieldsForm} />
		{:else}
			<div class="text-muted-foreground text-sm">
				<Alert variant="info" icon={Info}>
					{#snippet content({ Title, Description })}
						<Title class="font-bold">Info</Title>
						<Description class="mb-2">
							{m.json_configuration_is_edited_fields_are_disabled()}
						</Description>

						<Button variant="outline" onclick={() => form.jsonForm.reset()}>
							{m.reset_json_and_use_fields()}
						</Button>
					{/snippet}
				</Alert>
			</div>
		{/if}
	</div>

	<div class="flex min-w-0 shrink-0 grow basis-1 flex-col">
		<TestConfigJsonFormComponent form={form.jsonForm} />
	</div>
</div>
