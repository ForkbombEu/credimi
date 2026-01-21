// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CheckConfigFormEditor } from '$lib/start-checks-form/configure-checks-form/check-config-form-editor';
import type { BaseEditor } from '$start-checks-form/_utils';
import type { SuperForm, TaintedFields } from 'sveltekit-superforms';

import { yamlStringSchema } from '$lib/utils';
import { nanoid } from 'nanoid';
import { watch } from 'runed';
import { fromStore } from 'svelte/store';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod/v3';

import type { State } from '@/utils/types';

import { createForm } from '@/forms';

//

type CheckConfigJsonEditorProps = {
	code: string;
	editorDependency?: CheckConfigFormEditor;
};

type FormData = {
	code: string;
};

export class CheckConfigCodeEditor implements BaseEditor {
	public readonly superform: SuperForm<FormData>;
	private values: State<FormData>;
	private taintedState: State<TaintedFields<FormData> | undefined>;

	isTainted = $derived.by(() => this.taintedState.current?.code === true);
	isValid = $state(false);

	constructor(public readonly props: CheckConfigJsonEditorProps) {
		this.superform = createForm({
			adapter: zod(z.object({ code: yamlStringSchema })),
			initialData: { code: this.props.code },
			options: {
				id: nanoid(6)
			}
		});
		this.values = fromStore(this.superform.form);
		this.taintedState = fromStore(this.superform.tainted);
		this.effectValidateSuperform();
	}

	private effectValidateSuperform() {
		watch(
			() => this.values.current,
			() => {
				this.superform.validateForm({ update: false }).then(({ valid }) => {
					this.isValid = valid;
				});
			}
		);
	}

	getData() {
		return this.values.current.code;
	}

	reset() {
		this.superform.reset();
	}
}
