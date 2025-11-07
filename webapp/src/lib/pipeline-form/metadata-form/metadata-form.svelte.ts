// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createForm } from '@/forms';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';
import Component from './metadata-form.svelte';

//

type Metadata = z.infer<typeof metadataSchema>;

export class MetadataForm {
	#value: Partial<Metadata> = $state({});
	get value() {
		return this.#value;
	}

	readonly superform = createForm({
		adapter: zod(metadataSchema),
		onSubmit: async ({ form }) => {
			this.#value = form.data;
			this.isOpen = false;
		}
	});

	constructor() {
		$effect(() => {
			if (!this.isOpen) return;
			this.superform.form.update((data) => ({ ...data, ...this.#value }));
		});
	}

	isOpen = $state(false);
	readonly Component = Component;
}

const metadataSchema = z.object({
	description: z.string().min(3),
	published: z.boolean().default(false),
	name: z.string().min(3)
});
