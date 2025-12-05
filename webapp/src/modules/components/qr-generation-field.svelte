<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import QrFieldWrapper from '$lib/layout/qr-field-wrapper.svelte';
	import { generateDeeplinkFromYaml } from '$lib/utils';
	import { onMount } from 'svelte';
	import { fromStore, type Writable } from 'svelte/store';
	import { formFieldProxy } from 'sveltekit-superforms';

	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { QrCode } from '@/qr';

	//

	interface Props {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		form: SuperForm<any>;
		fieldName: string;
		label?: string;
		description?: string;
		placeholder?: string;
		successMessage?: string;
		loadingMessage?: string;
		enableStructuredErrors?: boolean;
	}

	let {
		form,
		fieldName,
		label = m.YAML_Configuration(),
		description = m.Provide_configuration_in_YAML_format(),
		placeholder = m.Run_the_code_to_generate_QR_code(),
		successMessage = m.Test_Completed_Successfully(),
		loadingMessage = m.Running_test()
	}: Props = $props();

	const fieldProxy = formFieldProxy(form, fieldName);
	const value = fromStore(fieldProxy.value as Writable<string>);
	const errors = fromStore(fieldProxy.errors);

	const hasErrors = $derived.by(() => {
		const count = errors.current?.length;
		return Boolean(count && count > 0);
	});

	onMount(() => {
		form.validate(fieldProxy.path, { update: true }).then(() => {
			const shouldRun = Boolean(value.current.trim()) && !hasErrors;
			if (shouldRun) startWorkflowTest(value.current);
		});
	});

	$effect(() => {
		if (value.current) workflowError = undefined;
	});

	//

	let workflowError = $state<unknown>();
	let isSubmittingWorkflow = $state(false);
	let generatedDeeplink = $state<string>();
	let workflowSteps = $state<unknown[]>();
	let workflowOutput = $state<unknown[]>();

	async function startWorkflowTest(yamlContent: string) {
		if (!yamlContent?.trim()) {
			workflowError = 'YAML configuration is required';
			return;
		}

		isSubmittingWorkflow = true;
		// Clear previous results
		generatedDeeplink = undefined;
		workflowSteps = undefined;
		workflowOutput = undefined;
		workflowError = undefined;

		try {
			const result = await generateDeeplinkFromYaml(yamlContent);
			generatedDeeplink = result.deeplink;
			workflowSteps = result.steps;
			workflowOutput = result.output;
		} catch (error) {
			workflowError = error;
		} finally {
			isSubmittingWorkflow = false;
		}
	}

	function getStepName(step: unknown, fallback: string): string {
		if (step && typeof step === 'object' && step !== null) {
			const obj = step as Record<string, unknown>;
			if (typeof obj.name === 'string') {
				return obj.name;
			}
		}
		return fallback;
	}

	const editorOutput = $derived(() => {
		if (isSubmittingWorkflow) {
			return loadingMessage;
		}

		if (workflowError) {
			return ''; // Error will be shown via the error prop
		}

		if (!generatedDeeplink && !workflowSteps && !workflowOutput) {
			return '';
		}

		let output = '';

		if (generatedDeeplink) {
			output += `✅ ${successMessage}\n\n`;
			output += `🔗 Generated Deeplink:\n${generatedDeeplink}\n\n`;
		}

		if (workflowSteps && workflowSteps.length > 0) {
			output += `📋 Workflow Execution Summary:\n`;
			output += `   Total Steps: ${workflowSteps.length}\n`;
			workflowSteps.forEach((step, index) => {
				const stepName = getStepName(step, `Step ${index + 1}`);
				const stepObj = step as Record<string, unknown>;
				const status = stepObj.status || stepObj.result || '✓';
				output += `   ${index + 1}. ${stepName} - ${status}\n`;
			});
			output += '\n';
		}

		if (workflowOutput && workflowOutput.length > 0) {
			output += `🧪 Test Results Summary:\n`;
			output += `   Total Tests: ${workflowOutput.length}\n`;
			workflowOutput.forEach((test, index) => {
				const testName = getStepName(test, `Test ${index + 1}`);
				const testObj = test as Record<string, unknown>;
				const status = testObj.status || testObj.result || 'PASSED';
				output += `   ${index + 1}. ${testName} - ${status}\n`;
			});
			output += '\n';
		}

		if (workflowOutput && workflowOutput.length > 0) {
			output += `📊 Detailed Results:\n`;
			output += JSON.stringify({ steps: workflowSteps, output: workflowOutput }, null, 2);
		}

		return output;
	});

	// Helper function for code editor error display
	const codeEditorErrorDisplay = $derived(() => {
		if (typeof workflowError === 'string') {
			return workflowError;
		}
		if (workflowError && typeof workflowError === 'object') {
			return JSON.stringify(workflowError, null, 2);
		}
		return undefined;
	});

	//

	const { constraints } = formFieldProxy(form, fieldName);
</script>

{#snippet error()}
	{#if workflowError && typeof workflowError === 'object' && 'summary' in workflowError}
		{@const error = workflowError}
		<div class="space-y-2 text-center">
			<div class="text-sm font-medium">{error.summary}</div>
		</div>
	{/if}
{/snippet}

<QrFieldWrapper {label} required={$constraints?.required}>
	<div class="flex max-w-full gap-4">
		<div class="w-0 grow">
			<CodeEditorField
				{form}
				name={fieldName}
				options={{
					lang: 'yaml',
					minHeight: 200,
					hideLabel: true,
					label,
					description,
					useOutput: true,
					output: editorOutput(),
					error: codeEditorErrorDisplay(),
					running: isSubmittingWorkflow,
					onRun: startWorkflowTest,
					canRun: !hasErrors
				}}
			/>
		</div>

		<div>
			<QrCode
				src={generatedDeeplink}
				class="size-60 rounded-md border"
				{placeholder}
				bind:isLoading={isSubmittingWorkflow}
				error={typeof workflowError === 'string' ? workflowError : undefined}
				hasStructuredError={!!(
					workflowError &&
					typeof workflowError === 'object' &&
					'summary' in workflowError
				)}
			>
				{#if workflowError && typeof workflowError === 'object' && 'summary' in workflowError}
					{@render error()}
				{/if}
			</QrCode>
			{#if generatedDeeplink}
				<div class="max-w-60 break-all pt-4 text-xs">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a class="hover:underline" href={generatedDeeplink} target="_self">
						{generatedDeeplink}
					</a>
				</div>
			{/if}
		</div>
	</div>
</QrFieldWrapper>
