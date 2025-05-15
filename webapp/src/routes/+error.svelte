<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import T from '@/components/ui-custom/t.svelte';
	import A from '@/components/ui-custom/a.svelte';
	import { dev } from '$app/environment';

	//

	const status = $derived(page.status);
	const message = $derived(page.error?.message);

	//

	type ErrorData = {
		status: number;
		image: string;
		title: string;
		description: string;
	};

	const data: ErrorData = $derived.by(() => {
		switch (status) {
			case 404:
				return {
					status: 404,
					image: '/404-computer.svg',
					title: 'Not Found',
					description: 'Whoops! This page doesn’t exist.'
				};
			case 503:
				return {
					status: 503,
					image: '/maintenance.svg',
					title: 'Service Unavailable',
					description:
						'We’re temporarily offline for maintenance.<br>Please try again later.'
				};
			default:
				return {
					status: 500,
					image: '/500.svg',
					title: 'Internal Error',
					description: 'Something went wrong.<br>Please try again later.'
				};
		}
	});
</script>

<div
	class="mx-auto flex min-h-screen max-w-4xl flex-col gap-4 p-8 pt-20 lg:flex-row lg:p-0 lg:pt-0"
>
	<div class="flex w-full flex-col items-center justify-center">
		<img src={data.image} alt={data.title} class="max-h-96" />
	</div>

	<div class="flex w-full flex-col items-center justify-center gap-4 text-center">
		<T tag="h3" class="text-primary-600">{data.status} {data.title}</T>
		<T tag="h2">{@html data.description}</T>
		{#if message && dev}
			<T class="font-mono text-xs text-gray-400">{message}</T>
		{/if}

		{#if status !== 503}
			<div class="flex flex-col items-center space-y-1 pt-8">
				<T class="text-gray-400">Here are some helpful links:</T>
				<ul class="flex gap-4">
					<li><A href="/">Home</A></li>
					<li><A href="/login">Login</A></li>
					<li><A href="/register">Register</A></li>
				</ul>
			</div>
		{/if}
	</div>
</div>
