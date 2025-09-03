<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { ComponentProps } from 'svelte';
	import type { Writable } from 'svelte/store';
	import type { SuperForm } from 'sveltekit-superforms';

	import { capitalize } from 'effect/String';
	import { fieldProxy, type FormPathLeaves } from 'sveltekit-superforms/client';

	import type { GenericRecord } from '@/utils/types';

	import * as Form from '@/components/ui/form';
	import { Switch } from '@/components/ui/switch';

	import type { FieldOptions } from './types';

	//

	interface Props {
		form: SuperForm<Data, any>;
		name: FormPathLeaves<Data, boolean>;
		options?: Partial<FieldOptions> & ComponentProps<typeof Switch>;
	}

	const { form, name, options = {} }: Props = $props();

	const value: Writable<boolean> = fieldProxy(form, name);
</script>

<Form.Field {form} {name}>
	<Form.Control>
		{#snippet children({ props })}
			<div class="flex items-center gap-2">
				<Switch {...props} checked={$value} onCheckedChange={(v) => ($value = v)} />
				<Form.Label>{options.label ?? capitalize(name)}</Form.Label>
			</div>
		{/snippet}
	</Form.Control>

	{#if options.description}
		<Form.Description>{options.description}</Form.Description>
	{/if}

	<Form.FieldErrors />
</Form.Field>
