<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends t.GenericRecord">
	import type { Snippet } from 'svelte';

	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import { form as _form, types as t } from '#';

	//

	type Props = {
		// eslint-disable-next-line no-undef
		form: _form.Instance<T>;
		children?: Snippet;
	};

	let { form }: Props = $props();
	form.attachSuperform();
</script>
