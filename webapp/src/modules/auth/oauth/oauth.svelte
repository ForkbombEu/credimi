<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ClientResponseError } from 'pocketbase';

	import { PUBLIC_POCKETBASE_URL } from '$env/static/public';
	import { PUBLIC_TURNSTILE_SITE_KEY } from '$env/static/public';
	import { nanoid } from 'nanoid';
	import { onMount } from 'svelte';

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
		requireCaptcha?: boolean;
	};

	const { hideOr = false, requireCaptcha = false }: Props = $props();

	//

	let error = $state<ClientResponseError>();
	let loading = $state(false);
	let captchaToken = $state('');
	let captchaError = $state('');
	let turnstileContainer = $state<HTMLDivElement>();
	let turnstileWidgetId = $state('');
	let turnstileScriptLoaded = $state(false);

	const authMethods = pb
		.collection('users')
		.listAuthMethods()
		.then((list) =>
			list.oauth2.providers.map((provider) => {
				return {
					displayName: provider.displayName,
					image: `${PUBLIC_POCKETBASE_URL}/_/images/oauth2/${provider.name}.svg`, // TODO - This won't work with `oidc2` for example
					initializer: async () => {
						if (requireCaptcha && !captchaToken) {
							captchaError = m.Please_complete_the_captcha();
							return;
						}
						captchaError = '';
						loading = true;
						try {
							const createData: Data<UsersRecord> = { name: nanoid(5) };
							const authData = await pb.collection('users').authWithOAuth2({
								provider: provider.name,
								createData,
								headers: requireCaptcha
									? { 'X-Turnstile-Token': captchaToken }
									: undefined
							});
							$currentUser = authData.record;
							goto('/my');
						} catch (e) {
							error = e as ClientResponseError;
							if (requireCaptcha) {
								captchaToken = '';
								captchaError = m.Please_complete_the_captcha();
							}
							if (requireCaptcha && turnstileWidgetId && window.turnstile) {
								window.turnstile.reset(turnstileWidgetId);
							}
						}
						loading = false;
					}
				};
			})
		);

	function renderTurnstile() {
		if (!requireCaptcha || !turnstileContainer || !window.turnstile || turnstileWidgetId) {
			return;
		}

		turnstileWidgetId = window.turnstile.render(turnstileContainer, {
			sitekey: PUBLIC_TURNSTILE_SITE_KEY,
			callback: (token: string) => {
				captchaToken = token;
				captchaError = '';
			},
			theme: 'auto'
		});
	}

	$effect(() => {
		if (turnstileScriptLoaded && turnstileContainer) {
			renderTurnstile();
		}
	});

	onMount(() => {
		if (!requireCaptcha) {
			return;
		}

		const existingScript = document.querySelector<HTMLScriptElement>(
			'script[src^="https://challenges.cloudflare.com/turnstile/v0/api.js"]'
		);

		const markTurnstileReady = () => {
			turnstileScriptLoaded = true;
			renderTurnstile();
		};

		if (window.turnstile) {
			markTurnstileReady();
			return;
		}

		if (existingScript) {
			existingScript.addEventListener('load', markTurnstileReady, { once: true });
			return;
		}

		const script = document.createElement('script');
		script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
		script.async = true;
		script.defer = true;
		script.onload = markTurnstileReady;
		document.head.appendChild(script);
	});
</script>

{#await authMethods then methods}
	{#if requireCaptcha && methods.length > 0}
		<div bind:this={turnstileContainer} class="mb-4 flex justify-center"></div>
	{/if}

	{#if requireCaptcha && captchaError}
		<p class="mb-4 text-sm text-red-600 dark:text-red-400">{captchaError}</p>
	{/if}

	{#each methods as method (method.displayName)}
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
			<p class="text-xs tracking-wide text-gray-400 uppercase">{m.or()}</p>
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
