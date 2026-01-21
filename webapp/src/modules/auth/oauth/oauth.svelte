<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClientResponseError } from 'pocketbase';

	import { PUBLIC_POCKETBASE_URL } from '$env/static/public';
	import { nanoid } from 'nanoid';

	import type { Data, UsersRecord } from '@/pocketbase/types';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import { Separator } from '@/components/ui/separator';
	import { goto, m } from '@/i18n';
	import { currentUser, pb } from '@/pocketbase';

	//

	type Props = {
		hideOr?: boolean;
	};

	const { hideOr = false }: Props = $props();

	//

	let error = $state<ClientResponseError>();
	let loading = $state(false);

	const authMethods = pb
		.collection('users')
		.listAuthMethods()
		.then((list) =>
			list.oauth2.providers.map((provider) => {
				return {
					displayName: provider.displayName,
					image: `${PUBLIC_POCKETBASE_URL}/_/images/oauth2/${provider.name}.svg`, // TODO - This won't work with `oidc2` for example
					initializer: async () => {
						loading = true;
						try {
							const createData: Data<UsersRecord> = { name: nanoid(5) };
							const authData = await pb.collection('users').authWithOAuth2({
								provider: provider.name,
								createData
							});
							$currentUser = authData.record;
							goto('/my');
						} catch (e) {
							error = e as ClientResponseError;
						}
						loading = false;
					}
				};
			})
		);
</script>

{#await authMethods then methods}
	{#each methods as method}
		<Button class="w-full" variant="outline" onclick={method.initializer}>
			<figure class="size-6 rounded-sm bg-white p-0.5">
				<img src={method.image} alt="{method.displayName} logo" />
			</figure>
			{m.Continue_with_oauthProvider({ oauthProvider: method.displayName })}
		</Button>
	{/each}

	{#if methods.length > 0 && !hideOr}
		<div class="flex items-center gap-2">
			<Separator class="grow basis-1" />
			<p class="text-xs uppercase tracking-wide text-gray-400">{m.or()}</p>
			<Separator class="grow basis-1" />
		</div>
	{/if}
{/await}

{#if error}
	{@const { message } = error}
	<Alert>
		{#snippet content({ Title, Description })}
			<Title>{m.Error()}</Title>
			<Description>{message}</Description>
		{/snippet}
	</Alert>
{/if}

{#if loading}
	<LoadingDialog />
{/if}
