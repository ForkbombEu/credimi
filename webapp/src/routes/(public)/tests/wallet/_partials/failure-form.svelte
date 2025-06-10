<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->
<script lang="ts">
	import { Form, SubmitButton, createForm } from '@/forms';
	import TextareaField from '@/forms/fields/textareaField.svelte';
	import type { SuperForm } from 'sveltekit-superforms/client';

	type Props = {
		failureForm: SuperForm<{ reason: string }, any>;
		age?: number;
	};

	const { failureForm }: Props = $props();
</script>

<div class="grow basis-1">
	<Form form={failureForm} hideRequiredIndicator class="space-y-2">
		<TextareaField
			form={failureForm}
			name="reason"
			options={{
				label: 'If something went wrong, please tell us what:'
			}}
		/>
		<SubmitButton class="w-full bg-red-600 hover:bg-red-700">Notify issue</SubmitButton>
	</Form>
</div>
