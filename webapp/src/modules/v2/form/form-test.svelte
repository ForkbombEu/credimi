<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends t.GenericRecord">
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import { form, types as t } from '@/v2';

	type Props = {
		// eslint-disable-next-line no-undef
		form: form.Form<T>;
	};

	// eslint-disable-next-line svelte/valid-compile
	let props: Props = $props();
	props.form.attachSuperform();
</script>
