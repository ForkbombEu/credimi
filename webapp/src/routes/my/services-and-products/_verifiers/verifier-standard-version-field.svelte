<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';
	import { stringProxy } from 'sveltekit-superforms/client';
	import * as Form from '@/components/ui/form';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';
	import type { VerifiersFormData } from '@/pocketbase/types';
	import { m } from '@/i18n';
	import type { SelectOption } from '@/components/ui-custom/utils';
	import { getStandardsAndVersionsFlatOptionsList } from '$lib/standards';
	import * as Select from '@/components/ui/select';
	import { fromStore, type Writable } from 'svelte/store';
	import { String } from 'effect';

	//

	interface Props {
		form: SuperForm<VerifiersFormData>;
	}

	const { form }: Props = $props();

	//

	const name = 'standard_and_version';
	const value = stringProxy(form, name, { empty: 'null' }) as Writable<string | null>;
	let valueState = fromStore(value);
	const { form: formData } = form;

	//

	let options: SelectOption<string>[] = $state([]);

	getStandardsAndVersionsFlatOptionsList().then((response) => {
		options = response;
	});

	//

	const SEP = ',';

	function getValue(): string[] {
		return (
			valueState.current
				?.split(SEP)
				.filter(String.isNonEmpty)
				.map((v) => v.trim()) ?? []
		);
	}

	function setValue(v: string[]) {
		valueState.current = v.join(SEP);
	}

	function getValueString(): string {
		const v = getValue();
		if (v.length === 0) return '';
		if (v.length === 1) return v[0];
		else return `${v.length} values`;
	}
</script>

<Form.Field {form} {name}>
	<FieldWrapper
		field={name}
		options={{
			label: m.Compliance_standard(),
			description: `${m.eg()}: OpenID4VP Verifier - Draft 23`
		}}
	>
		{#snippet children({ props })}
			<Select.Root type="multiple" bind:value={getValue, setValue}>
				<Select.Trigger {...props}>
					{#if getValue().length}
						{getValueString()}
					{:else}
						{m.Select_a_standard_and_version()}
					{/if}
				</Select.Trigger>

				<Select.Content>
					{#each options as item}
						<Select.Item value={item.value} disabled={item.disabled}>
							{item.label ?? item.value}
						</Select.Item>
					{:else}
						<Select.Item class="flex justify-center [&>span]:hidden" disabled value="">
							{m.No_options_available()}
						</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		{/snippet}
	</FieldWrapper>
</Form.Field>
