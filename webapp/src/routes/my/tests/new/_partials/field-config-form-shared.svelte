<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form } from '@/forms';
	import { zod } from 'sveltekit-superforms/adapters';
	import { createInitialDataFromFields, createTestVariablesFormSchema } from './logic';
	import { Store, watch } from 'runed';
	import FieldConfigToFormField from './field-config-to-form-field.svelte';
	import { pipe, Tuple, Record } from 'effect';
	import { nanoid } from 'nanoid';
	import type { FieldConfig } from './tests-configs-form/types';

	//

	type Props = {
		fields: FieldConfig[];
		initialData?: Record<string, unknown>;
		onUpdate?: (form: Record<string, unknown>) => void;
	};

	let { fields, onUpdate }: Props = $props();

	//

	const form = createForm({
		adapter: zod(createTestVariablesFormSchema(fields)),
		initialData: createInitialDataFromFields(fields),
		options: {
			id: nanoid(6)
		}
	});

	const { form: formData, validate } = form;
	const formState = new Store(formData);

	watch(
		() => formState.current,
		() => {
			getValidFormData().then((data) => {
				onUpdate?.(data);
			});
		}
	);

	async function getValidFormData() {
		const results = await pipe(
			fields.map(({ CredimiID: id }) =>
				validate(id, { update: false }).then((result) =>
					Tuple.make(id, !Boolean(result?.length))
				)
			),
			(results) => Promise.all(results).then(Record.fromEntries)
		);
		const validData = pipe(
			formState.current,
			Record.filter((_, id) => results[id])
		);
		return validData;
	}
</script>

<Form {form} hide={['submit_button']} hideRequiredIndicator>
	{#each fields as config}
		<FieldConfigToFormField {config} {form} />
	{/each}
</Form>
