<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { WithLabel } from '$pipeline-form/steps/_partials/index.js';

	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import Select from '@/components/ui-custom/select.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { m } from '@/i18n';

	import {
		JSON_PARSE_STRUCT_TYPES,
		type JsonParseStepForm
	} from './json-parse-step-form.svelte.js';

	//

	let { self: form }: SelfProp<JsonParseStepForm> = $props();

	const structTypes = JSON_PARSE_STRUCT_TYPES.map((type) => ({ value: type, label: type }));
</script>

<div class="space-y-6 p-4">
	<WithLabel label={m.Raw_JSON()} required>
		<CodeEditor lang="json" bind:value={form.data.rawJSON} />
	</WithLabel>

	<WithLabel label={m.Struct_type()} required>
		<Select items={structTypes} bind:value={form.data.struct_type} />
	</WithLabel>

	{#if form.intent === 'add'}
		<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
			<T>{m.Add_step()}</T>
		</Button>
	{/if}
</div>
