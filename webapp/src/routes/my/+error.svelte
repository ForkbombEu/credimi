<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';

	import A from '@/components/ui-custom/a.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	const status = $derived(page.status);
	const title = $derived(status === 404 ? m.Not_Found() : m.Internal_Error());
	const image = $derived(status === 404 ? '/404-computer.svg' : '/500.svg');
</script>

<div
	class="mx-auto flex min-h-screen max-w-4xl flex-col gap-4 p-8 pt-20 lg:flex-row lg:p-0 lg:pt-0"
>
	<div class="flex w-full flex-col items-center justify-center">
		<img src={image} alt={title} />
	</div>
	<div class="flex w-full flex-col items-center justify-center gap-4">
		<T tag="h2" class="text-primary-600">{status} • {title}</T>
		<A href="/">{m.Come_back_home()}</A>
	</div>
</div>
