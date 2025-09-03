<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export const currentEmail = $state({ value: '' });
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { page } from '$app/state';

	import type { Link } from '@/components/types';

	import Oauth from '@/auth/oauth/oauth.svelte';
	import A from '@/components/ui-custom/a.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { featureFlags } from '@/features';
	import { m } from '@/i18n';
	import { localizeHref } from '@/i18n';

	//

	interface Props {
		children?: Snippet;
	}

	let { children }: Props = $props();

	const modes: Link[] = [
		{
			href: localizeHref('/login'),
			title: m.Email_and_password()
		},
		{
			href: localizeHref('/login/webauthn'),
			title: m.Webauthn()
		}
	];
</script>

<T tag="h4">Log in</T>

{#if $featureFlags.OAUTH}
	<Oauth></Oauth>
{/if}

{#if $featureFlags.WEBAUTHN}
	<div class="space-y-2">
		<T tag="small" class="text-gray-500">{m.Choose_your_authentication_method()}</T>
		<div class="flex items-center overflow-hidden rounded-md border">
			{#each modes as { href, title }}
				{@const isActive = page.url.pathname === href}
				<Button
					variant={isActive ? 'secondary' : 'outline'}
					{href}
					class="grow rounded-none border-none"
				>
					{title}
				</Button>
			{/each}
		</div>
	</div>
{/if}

<div class="pt-4">
	{@render children?.()}
</div>

<div class="flex flex-col gap-4 space-y-2">
	<Separator />

	<T class="self-center text-gray-500 dark:text-gray-400" tag="small">
		{m.Dont_have_an_account()}
		<A href="/register">{m.Register_here()}</A>
	</T>
</div>
