<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import { onMount } from 'svelte';
	import { fromStore } from 'svelte/store';
	import { stringProxy, type SuperForm } from 'sveltekit-superforms';
	import { z } from 'zod';

	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import type { GenericRecord } from '@/utils/types';

	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import QrStateful from '@/qr/qr-stateful.svelte';

	//

	interface Props {
		form: SuperForm<{ deeplink: string; yaml: string }>;
	}

	let { form }: Props = $props();

	const yamlField = fromStore(stringProxy(form, 'yaml', { empty: 'undefined' }));

	onMount(() => {
		if (yamlField.current && yamlField.current.trim()) {
			startComplianceTest(yamlField.current);
		}
	});

	//

	let workflowError = $state<string>();
	let isSubmittingCompliance = $state(false);
	let credentialOffer = $state<string>();
	let workflowSteps = $state<unknown[]>();
	let workflowOutput = $state<unknown[]>();

	async function startComplianceTest(yamlContent: string) {
		if (!yamlContent.trim() || !yamlContent) {
			workflowError = 'YAML configuration is required';
			return;
		}

		isSubmittingCompliance = true;
		// Clear previous results
		credentialOffer = undefined;
		workflowSteps = undefined;
		workflowOutput = undefined;
		workflowError = undefined;

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
			},
			requestKey: null
		});
		console.log(res);
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

<div class="flex max-w-full gap-4 pt-6">
	<div class="w-0 grow">
		<CodeEditorField
			{form}
			name="yaml"
			options={{
				lang: 'yaml',
				minHeight: 200,
				label: m.YAML_Configuration(),
				description: 'Provide the credential configuration in YAML format',
				useOutput: true,
				output: editorOutput(),
				error: workflowError || undefined,
				running: isSubmittingCompliance,
				onRun: startComplianceTest
			}}
		/>
	</div>

	<div class="pt-8">
		<QrStateful
			src={credentialOffer}
			class="size-60 rounded-md border"
			placeholder="Run the code to generate a QR code"
			bind:isLoading={isSubmittingCompliance}
			bind:error={workflowError}
		/>
		{#if credentialOffer}
			<div class="max-w-60 break-all pt-4 text-xs">
				<a href={credentialOffer} target="_self">{credentialOffer}</a>
			</div>
		{/if}
	</div>
</div>
