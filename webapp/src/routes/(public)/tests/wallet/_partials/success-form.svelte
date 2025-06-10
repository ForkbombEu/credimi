<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->
<script lang="ts">
	import { Form, SubmitButton, createForm } from '@/forms';
	import { Label } from '@/components/ui/label';
	import type { SuperForm } from 'sveltekit-superforms';

	type Props = {
		successForm: SuperForm<{}, any>;
	};
	const { successForm }: Props = $props();
</script>

<div class="grow basis-1">
	<Form form={successForm}>
		<div class="space-y-2">
			<Label for="success">If the test succeeded:</Label>
			<SubmitButton id="success" class="w-full bg-green-600 hover:bg-green-700">
				Confirm test success
			</SubmitButton>
		</div>
	</Form>
</div>
