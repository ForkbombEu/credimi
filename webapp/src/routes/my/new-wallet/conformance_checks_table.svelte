<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export const confChecksTable = z.object({
		runId: z.string(),
		standard: z.string(),
		test: z.string()
	});
	export type ConfChecksTable = z.infer<typeof confChecksTable>;
</script>

<script lang="ts" generics="Data extends GenericRecord & { conformance_checks: string[] }">
	import type { FormPath, SuperForm } from 'sveltekit-superforms';

	import { z } from 'zod';

	import type { FieldOptions } from '@/forms/fields/types';
	import type { GenericRecord } from '@/utils/types';

	import { Checkbox } from '@/components/ui/checkbox';
	import * as Form from '@/components/ui/form';
	import * as Table from '@/components/ui/table/index.js';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';

	type Props = {
		data: ConfChecksTable[];
		form: SuperForm<Data>;
		name: FormPath<Data>;
		options: Partial<FieldOptions>;
	};

	let { data, form, name, options }: Props = $props();

	const { form: formData } = form;
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} options={{ label: options.label, description: options.description }}>
		{#snippet children({ props })}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head></Table.Head>
						<Table.Head>Run ID</Table.Head>
						<Table.Head>Standard</Table.Head>
						<Table.Head>Test</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each data as row}
						<Table.Row>
							<Table.Cell
								><Checkbox
									onCheckedChange={(e) => {
										if (e) {
											$formData.conformance_checks = [
												...$formData.conformance_checks,
												row.runId
											];
										} else {
											$formData.conformance_checks =
												$formData.conformance_checks.filter(
													(id) => id !== row.runId
												);
										}
									}}
								/></Table.Cell
							>
							<Table.Cell>{row.runId}</Table.Cell>
							<Table.Cell>{row.standard}</Table.Cell>
							<Table.Cell>{row.test}</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/snippet}
	</FieldWrapper>
</Form.Field>
