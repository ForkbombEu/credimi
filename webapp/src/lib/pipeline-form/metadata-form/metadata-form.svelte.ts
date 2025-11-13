// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createForm } from '@/forms';
import { tick } from 'svelte';
import type { SuperForm } from 'sveltekit-superforms';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';
import Component from './metadata-form.svelte';

//

type Props = {
	initialData?: Metadata;
	onSubmit?: () => void | Promise<void>;
};

export type Metadata = z.infer<typeof metadataSchema>;

export class MetadataForm {
	constructor(private props: Props) {
		this.#value = props.initialData;
	}

	#value = $state<Metadata>();
	get value() {
		return this.#value;
	}

	superform: SuperForm<Metadata> | undefined;

	mountForm() {
		this.superform = createForm({
			adapter: zod(metadataSchema),
			initialData: this.#value,
			onSubmit: async ({ form }) => {
				this.#value = form.data;
				await tick();
				await this.props.onSubmit?.();
				this.isOpen = false;
			}
		});
		return this.superform;
	}

	#isValid = $state(false);
	get isValid() {
		return this.#isValid;
	}

	isOpen = $state(false);
	readonly Component = Component;
}

export const metadataSchema = z.object({
	description: z.string().min(3),
	published: z.boolean().optional(),
	name: z.string().min(3)
});
