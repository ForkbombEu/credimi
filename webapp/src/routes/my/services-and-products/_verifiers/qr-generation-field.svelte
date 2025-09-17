<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { fromStore } from 'svelte/store';
	import { stringProxy, type SuperForm } from 'sveltekit-superforms';
	import { z } from 'zod';

	import Label from '@/components/ui/label/label.svelte';
	import { CodeEditorField } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import QrStateful from '@/qr/qr-stateful.svelte';

	//

	interface Props {
		form: SuperForm<{ deeplink: string; yaml?: string }>;
	}

	let { form }: Props = $props();

	const deeplinkField = fromStore(stringProxy(form, 'deeplink', { empty: 'undefined' }));

	onMount(() => {
		if (deeplinkField.current && deeplinkField.current.trim()) {
			startVerificationTest(deeplinkField.current);
		}
	});

	$effect(() => {
		if (deeplinkField.current) {
			workflowError = undefined;
		}
	});

	//

	let workflowError = $state<string>();
	let isSubmittingVerification = $state(false);
	let verificationDeeplink = $state<string>();
	let workflowSteps = $state<unknown[]>();
	let workflowOutput = $state<unknown[]>();

	async function startVerificationTest(yamlContent: string) {
		if (!yamlContent.trim() || !yamlContent) {
			workflowError = 'YAML configuration is required';
			return;
		}

		isSubmittingVerification = true;
		// Clear previous results
		verificationDeeplink = undefined;
		workflowSteps = undefined;
		workflowOutput = undefined;
		workflowError = undefined;

		try {
			const result = await processYamlAndExtractCredentialOffer(yamlContent);
			verificationDeeplink = result.credentialOffer;
			workflowSteps = result.steps;
			workflowOutput = result.output;
		} catch (error) {
			console.error('Failed to start compliance test:', error);
			workflowError = error instanceof Error ? error.message : String(error);
		} finally {
			isSubmittingVerification = false;
		}
	}

	async function processYamlAndExtractCredentialOffer(yaml: string) {
		const res = await pb.send('api/credentials_issuers/get-deeplink', {
			method: 'POST',
			body: {
				yaml
			},
			requestKey: null
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
		if (isSubmittingVerification) {
			return 'Running verification test...';
		}

		if (workflowError) {
			return ''; // Error will be shown via the error prop
		}

		if (!verificationDeeplink && !workflowSteps && !workflowOutput) {
			return '';
		}

		let output = '';

		if (verificationDeeplink) {
			output += `✅ Verification Test Completed Successfully!\n\n`;
			output += `🔗 Verification Deeplink Generated:\n${verificationDeeplink}\n\n`;
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

<div class="flex max-w-full gap-4">
	<div class="w-0 grow">
		<Label>Deeplink</Label>
		<CodeEditorField
			{form}
			name="deeplink"
			options={{
				lang: 'yaml',
				minHeight: 200,
				label: m.YAML_Configuration(),
				description:
					'Provide the verification configuration in YAML format. You can upload a file or paste/edit it directly.',
				useOutput: true,
				output: editorOutput(),
				error: workflowError || undefined,
				running: isSubmittingVerification,
				onRun: startVerificationTest
			}}
		/>
	</div>

	<div>
		<QrStateful
			src={verificationDeeplink}
			class="size-60 rounded-md border"
			placeholder="Run the code to generate a QR code"
			bind:isLoading={isSubmittingVerification}
			bind:error={workflowError}
		/>
		{#if verificationDeeplink}
			<div class="max-w-60 break-all pt-4 text-xs">
				<a class="hover:underline" href={verificationDeeplink} target="_self"
					>{verificationDeeplink}</a
				>
			</div>
		{/if}
	</div>
</div>
