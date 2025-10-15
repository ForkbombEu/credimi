<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export const ConformanceCheckSchema = z.object({
		runId: z.string(),
		standard: z.string(),
		test: z.string(),
		workflowId: z.string(),
		status: z.string()
	});
	export type ConformanceCheck = z.infer<typeof ConformanceCheckSchema>;
</script>

<script
	lang="ts"
	generics="Data extends GenericRecord & { conformance_checks?: ConformanceCheck[] }"
>
	import type { FormPath, SuperForm } from 'sveltekit-superforms';

	import { fetchWorkflows, getWorkflowMemo } from '$lib/workflows';
	import { z } from 'zod';

	import type { FieldOptions } from '@/forms/fields/types';
	import type { GenericRecord } from '@/utils/types';

	import { Checkbox } from '@/components/ui/checkbox';
	import * as Form from '@/components/ui/form';
	import * as Table from '@/components/ui/table/index.js';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';

	//

	type Props = {
		form: SuperForm<Data>;
		name: FormPath<Data>;
		options: Partial<FieldOptions>;
	};

	let { form, name, options }: Props = $props();
	const { form: formData } = form;

	//

	const tableData: Promise<ConformanceCheck[]> = fetchWorkflows().then((res) => {
		if (res instanceof Error) throw res;

		const memos = res.map((w) => getWorkflowMemo(w));
		const checks: ConformanceCheck[] = [];

		// Listing only checks that have a memo
		for (const [index, memo] of memos.entries()) {
			if (!memo) continue;
			const w = res[index];
			checks.push({
				runId: w.runId,
				standard: memo.standard,
				test: memo.test,
				workflowId: w.id,
				status: w.status ?? ''
			});
		}

		return checks;
	});
</script>

{#await tableData then data}
	<Form.Field {form} {name}>
		<FieldWrapper
			field={name}
			options={{ label: options.label, description: options.description }}
		>
			{#snippet children()}
				<Table.Root class="!rounded-md border">
					<Table.Header>
						<Table.Row>
							<Table.Head></Table.Head>
							<Table.Head>Status</Table.Head>
							<Table.Head>Standard</Table.Head>
							<Table.Head>Test</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each data as row}
							<Table.Row>
								<Table.Cell>
									<Checkbox
										checked={$formData.conformance_checks?.some(
											(item) => item.runId === row.runId
										)}
										onCheckedChange={(e) => {
											if (e) {
												$formData.conformance_checks = [
													...($formData.conformance_checks ?? []),
													row
												];
											} else {
												$formData.conformance_checks = (
													$formData.conformance_checks ?? []
												).filter((item) => item.runId !== row.runId);
											}
										}}
									/>
								</Table.Cell>
								<Table.Cell>{row.status}</Table.Cell>
								<Table.Cell>{row.standard}</Table.Cell>
								<Table.Cell>{row.test}</Table.Cell>
							</Table.Row>
						{:else}
							<Table.Row>
								<Table.Cell colspan={4} class="text-center">
									No conformance checks found
								</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			{/snippet}
		</FieldWrapper>
	</Form.Field>
{/await}
