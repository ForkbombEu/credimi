// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createForm } from '@/forms';
import { z } from 'zod';
import { formatJson, stringifiedObjectSchema, type BaseForm } from '$lib/start-checks-form/_utils';
import { zod } from 'sveltekit-superforms/adapters';
import type { SuperForm, TaintedFields } from 'sveltekit-superforms';
import { nanoid } from 'nanoid';
import type { State } from '@/utils/types';
import { fromStore } from 'svelte/store';
import type { TestConfigFieldsForm } from '$lib/start-checks-form/test-config-fields-form';
import { watch } from 'runed';

//

type TestConfigJsonFormProps = {
	json: string;
	formDependency?: TestConfigFieldsForm;
};

type FormData = {
	json: string;
};

export class TestConfigJsonForm implements BaseForm {
	superform: SuperForm<FormData>;
	values: State<FormData>;

	private taintedState: State<TaintedFields<FormData> | undefined>;
	isTainted = $derived.by(() => this.taintedState.current?.json === true);

	isValid = $state(false);

	constructor(public readonly props: TestConfigJsonFormProps) {
		this.superform = createForm({
			adapter: zod(z.object({ json: stringifiedObjectSchema })),
			initialData: { json: formatJson(this.props.json) },
			options: {
				id: nanoid(6)
			}
		});
		this.values = fromStore(this.superform.form);
		this.taintedState = fromStore(this.superform.tainted);
		this.effectValidateForm();
	}

	effectValidateForm() {
		watch(
			() => this.values.current,
			() => {
				this.superform.validateForm({ update: false }).then(({ valid }) => {
					this.isValid = valid;
				});
			}
		);
	}

	getFormData() {
		return {
			json: this.values.current.json
		};
	}

	reset() {
		this.superform.reset();
	}
}
