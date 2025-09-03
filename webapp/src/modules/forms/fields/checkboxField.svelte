<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { GenericRecord } from '@/utils/types';
</script>

<script lang="ts" generics="Data extends GenericRecord">
	import type { ComponentProps, Snippet } from 'svelte';
	import type { Writable } from 'svelte/store';
	import type { SuperForm } from 'sveltekit-superforms';

	import { capitalize } from 'effect/String';
	import { fieldProxy, type FormPathLeaves } from 'sveltekit-superforms/client';

	import { Checkbox } from '@/components/ui/checkbox';
	import * as Form from '@/components/ui/form';

	import type { FieldOptions } from './types';

	import RequiredIndicator from '../components/requiredIndicator.svelte';

	//

	interface Props {
		form: SuperForm<Data, any>;
		name: FormPathLeaves<Data, boolean>;
		options?: Partial<FieldOptions> & ComponentProps<typeof Checkbox>;
		children?: Snippet;
	}

	const { form, name, options = {}, children: childrenSnippet }: Props = $props();

	const value: Writable<boolean> = $derived(fieldProxy(form, name));
</script>

<Form.Field {form} {name}>
	<Form.Control>
		{#snippet children({ props })}
			<div class="flex items-center gap-2">
				<Checkbox {...props} checked={$value} onCheckedChange={(v) => ($value = v)} />

				<Form.Label>
					{#if childrenSnippet}
						{@render childrenSnippet()}
					{:else}
						{options.label ?? capitalize(name)}
					{/if}
					<RequiredIndicator field={name} />
				</Form.Label>
			</div>
		{/snippet}
	</Form.Control>

	{#if options.description}
		<Form.Description>{options.description}</Form.Description>
	{/if}

	<Form.FieldErrors />
</Form.Field>
