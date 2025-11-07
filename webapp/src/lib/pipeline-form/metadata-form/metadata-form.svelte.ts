// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createForm } from '@/forms';
import { fromStore } from 'svelte/store';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';
import Component from './metadata-form.svelte';

//

export class MetadataForm {
	readonly superform = createForm({
		adapter: zod(
			z.object({
				description: z.string().min(3),
				published: z.boolean().default(false),
				name: z.string().min(3)
			})
		),
		onSubmit: async ({ form }) => {
			this.isOpen = false;
		}
	});

	private readonly formState = fromStore(this.superform.form);
	readonly value = $derived(this.formState.current);

	isOpen = $state(false);
	readonly Component = Component;
}
