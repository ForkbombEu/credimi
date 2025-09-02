<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ButtonProps } from '@/components/ui/button';

	import * as Form from '@/components/ui/form';
	import { getFormContext } from '@/forms';

	import { FORM_ERROR_PATH } from '../form';

	//

	const { children, ...props }: ButtonProps = $props();

	const { form } = $derived(getFormContext());
	const { allErrors } = $derived(form);
	const formHasErrors = $derived($allErrors.filter((e) => e.path != FORM_ERROR_PATH).length > 0);
</script>

<Form.Button {...props} disabled={formHasErrors}>
	{@render children?.()}
</Form.Button>
