<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends Record<string, unknown>">
	import { form } from '@/v2';

	type Props = form.Config<T> & {
		onReady?: (form: form.Form<T>) => void;
	};

	// eslint-disable-next-line svelte/valid-compile
	let props: Props = $props();

	const f = new form.Form(props);
	props.onReady?.(f);
</script>
