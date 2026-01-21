<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { FileIcon, LogOut, SquareArrowOutUpRight } from '@lucide/svelte';

	import AuthLayout from '@/auth/authLayout.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu/index.js';
	import { getUserDidUrl } from '@/did';
	import { featureFlags } from '@/features';
	import { localizeHref, m } from '@/i18n';
	import BaseLanguageSelect from '@/i18n/baseLanguageSelect.svelte';
	import LanguageSelect from '@/i18n/languageSelect.svelte';
	import { currentUser } from '@/pocketbase';
	interface Props {
		children?: Snippet;
	}

	let { children }: Props = $props();
</script>

<AuthLayout>
	{#snippet topbarRight()}
		{#if $featureFlags.AUTH && $currentUser}
			<p class="mr-4">
				{m.hello()}, <span class="font-semibold">{$currentUser?.email}</span>
			</p>

			<DropdownMenu.Root>
				<DropdownMenu.Trigger class="rounded-md border p-1">
					<UserAvatar class="size-9" />
				</DropdownMenu.Trigger>

				<DropdownMenu.Content>
					<DropdownMenu.Group>
						<DropdownMenu.Sub>
							<BaseLanguageSelect>
								{#snippet trigger({ icon, text })}
									<DropdownMenu.SubTrigger>
										<Icon src={icon} />
										{text}
									</DropdownMenu.SubTrigger>
								{/snippet}
								{#snippet languages({ languages })}
									<DropdownMenu.SubContent>
										{#each languages as { name, flag, href, hreflang }}
											<DropdownMenu.Item>
												{#snippet child({ props })}
													<a
														{...props}
														href={localizeHref(href)}
														{hreflang}
													>
														<span>{flag}</span>
														<span>{name}</span>
													</a>
												{/snippet}
											</DropdownMenu.Item>
										{/each}
									</DropdownMenu.SubContent>
								{/snippet}
							</BaseLanguageSelect>
						</DropdownMenu.Sub>

						{#if $featureFlags.DID}
							{#await getUserDidUrl() then url}
								<DropdownMenu.Item>
									{#snippet child({ props })}
										<a
											{...props}
											href={url ? localizeHref(url) : undefined}
											target="_blank"
										>
											<Icon src={FileIcon} />
											{m.my_DID()}
											<Icon src={SquareArrowOutUpRight} class="opacity-50" />
										</a>
									{/snippet}
								</DropdownMenu.Item>
							{/await}
						{/if}

						<DropdownMenu.Item>
							{#snippet child({ props })}
								<a
									{...props}
									href={localizeHref('/logout')}
									data-sveltekit-preload-data="off"
								>
									<Icon src={LogOut} />
									{m.Sign_out()}
								</a>
							{/snippet}
						</DropdownMenu.Item>
					</DropdownMenu.Group>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{:else}
			<div class="mr-4">Hello!</div>
			<LanguageSelect />
		{/if}
	{/snippet}

	{@render children?.()}
</AuthLayout>
