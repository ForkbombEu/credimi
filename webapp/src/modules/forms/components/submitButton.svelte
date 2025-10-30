<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { fromStore } from 'svelte/store';

	import type { ButtonProps } from '@/components/ui/button';

	import * as Form from '@/components/ui/form';
	import { getFormContext } from '@/forms';

	import { FORM_ERROR_PATH } from '../form';

	//

	const { children, disabled = false, ...props }: ButtonProps = $props();

	const { form } = $derived(getFormContext());
	const { allErrors } = $derived(form);
	const formHasErrors = $derived($allErrors.filter((e) => e.path != FORM_ERROR_PATH).length > 0);

	// svelte-ignore state_referenced_locally
	const submitting = fromStore(form.submitting);
</script>

<Form.Button {...props} disabled={formHasErrors || submitting.current || disabled}>
	{@render children?.()}
</Form.Button>
