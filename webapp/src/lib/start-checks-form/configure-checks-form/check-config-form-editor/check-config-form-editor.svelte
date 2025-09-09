<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Form } from '@/forms';

	import type { CheckConfigFormEditor } from './check-config-form-editor.svelte.js';

	import ConfigFormInput from './check-config-form-input.svelte';

	//

	type Props = {
		editor: CheckConfigFormEditor;
	};

	let { editor }: Props = $props();
</script>

<Form form={editor.superform} hide={['submit_button']} class="space-y-6" hideRequiredIndicator>
	{#each editor.props.fields as field}
		<ConfigFormInput {field} form={editor.superform} />
	{/each}
</Form>
