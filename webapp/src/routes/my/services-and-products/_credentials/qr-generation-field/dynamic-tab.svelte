<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';

	import { yaml as yamlLang } from '@codemirror/lang-yaml';
	import { fromStore } from 'svelte/store';
	import { stringProxy } from 'sveltekit-superforms';
	import { z } from 'zod';

	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import type { GenericRecord } from '@/utils/types';

	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string>;
		isSubmittingCompliance: boolean;
		credentialOffer: string | null;
		workflowSteps: unknown[] | null;
		workflowOutput: unknown[] | null;
		workflowError: string | null;
	}

	let {
		form,
		name,
		isSubmittingCompliance = $bindable(),
		credentialOffer = $bindable(),
		workflowSteps = $bindable(),
		workflowOutput = $bindable(),
		workflowError = $bindable()
	}: Props = $props();

	const { form: formData } = $derived(form);
	const fieldProxy = $derived(stringProxy(formData, name, { empty: 'undefined' }));
	const fieldState = $derived(fromStore(fieldProxy));

	const currentYamlValue = $derived(() => {
		return fieldState.current || '';
	});

	let hasInitialized = $state(false);

	$effect(() => {
		const yamlValue = currentYamlValue();
		if (
			yamlValue &&
			yamlValue.trim() &&
			!credentialOffer &&
			!isSubmittingCompliance &&
			!hasInitialized
		) {
			startComplianceTest(yamlValue);
		}
		if (!hasInitialized) {
			hasInitialized = true;
		}
	});

	async function startComplianceTest(yamlContent: string) {
		if (!yamlContent.trim()) {
			workflowError = 'YAML configuration is required';
			return;
		}

		isSubmittingCompliance = true;
		// Clear previous results
		credentialOffer = null;
		workflowSteps = null;
		workflowOutput = null;
		workflowError = null;

		try {
			const result = await processYamlAndExtractCredentialOffer(yamlContent);
			credentialOffer = result.credentialOffer;
			workflowSteps = result.steps;
			workflowOutput = result.output;
		} catch (error) {
			console.error('Failed to start compliance test:', error);
			workflowError = error instanceof Error ? error.message : String(error);
		} finally {
			isSubmittingCompliance = false;
		}
	}

	async function processYamlAndExtractCredentialOffer(yaml: string) {
		const res = await pb.send('api/credentials_issuers/get-credential-deeplink', {
			method: 'POST',
			body: {
				yaml
			}
		});
		const responseSchema = z.object({
			credentialOffer: z.string(),
			steps: z.array(z.unknown()),
			output: z.array(z.unknown())
		});
		return responseSchema.parse(res);
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
		if (isSubmittingCompliance) {
			return 'Running compliance test...';
		}

		if (workflowError) {
			return ''; // Error will be shown via the error prop
		}

		if (!credentialOffer && !workflowSteps && !workflowOutput) {
			return '';
		}

		let output = '';

		if (credentialOffer) {
			output += `✅ Compliance Test Completed Successfully!\n\n`;
			output += `🔗 Credential Offer Generated:\n${credentialOffer}\n\n`;
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
</script>

<div class="space-y-4">
	<CodeEditorField
		{form}
		{name}
		options={{
			lang: yamlLang(),
			minHeight: 200,
			label: m.YAML_Configuration(),
			description: 'Provide the credential configuration in YAML format',
			useOutput: true,
			output: editorOutput(),
			error: workflowError || undefined,
			running: isSubmittingCompliance,
			onRun: (yamlContent: string) => {
				if (yamlContent && yamlContent.trim()) {
					startComplianceTest(yamlContent);
				}
			}
		}}
	/>
</div>
