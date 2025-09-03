<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ControlAttrs } from 'formsnap';
	import type { Snippet } from 'svelte';

	import { capitalize } from 'effect/String';

	import * as Form from '@/components/ui/form';
	import RequiredIndicator from '@/forms/components/requiredIndicator.svelte';

	import type { FieldOptions } from '../types';

	interface Props {
		field: string;
		options?: Partial<FieldOptions>;
		children?: Snippet<[{ props: ControlAttrs }]>;
	}

	const { field, options = {}, children: child }: Props = $props();
</script>

<Form.Control>
	{#snippet children({ props })}
		{#if !options.hideLabel}
			{#if !options.labelRight}
				{@render label()}
			{:else}
				<div class="flex items-center justify-between gap-4">
					{@render label()}
					{@render options.labelRight?.()}
				</div>
			{/if}
		{/if}

		{@render child?.({ props })}
	{/snippet}
</Form.Control>

{#if options.description}
	<Form.Description>{options.description}</Form.Description>
{/if}

<Form.FieldErrors />

{#snippet label()}
	<Form.Label>
		{options.label ?? capitalize(field)}
		<RequiredIndicator {field} />
	</Form.Label>
{/snippet}
