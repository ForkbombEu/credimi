<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { form, types as t } from '@/v2';

	export type Props<T extends t.GenericRecord> = form.Config<T> & {
		onReady?: (form: form.Form<T>) => void;
	};
</script>

<script lang="ts" generics="T extends Record<string, unknown>">
	// eslint-disable-next-line svelte/valid-compile, no-undef
	let props: Props<T> = $props();

	const f = new form.Form(props);

	$effect(() => {
		props.onReady?.(f);
	});
</script>
