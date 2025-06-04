// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { CheckConfigJsonEditor } from '$lib/start-checks-form/configure-checks-form/check-config-json-editor';
import {
	DependentCheckConfigFormEditor,
	type DependentCheckConfigFormEditorProps
} from '$lib/start-checks-form/configure-checks-form/check-config-form-editor';
import type { BaseEditor } from '../../_utils';

//

type CheckConfigEditorProps = DependentCheckConfigFormEditorProps & {
	id: string;
	json: string;
};

export type CheckConfigEditorMode = 'json' | 'form';

export class CheckConfigEditor implements BaseEditor {
	public readonly jsonEditor: CheckConfigJsonEditor;
	public readonly formEditor: DependentCheckConfigFormEditor;

	mode: CheckConfigEditorMode = $derived.by(() => (this.jsonEditor.isTainted ? 'json' : 'form'));

	isValid = $derived.by(() =>
		this.mode === 'json' ? this.jsonEditor.isValid : this.formEditor.isValid
	);

	constructor(public readonly props: CheckConfigEditorProps) {
		this.formEditor = new DependentCheckConfigFormEditor({
			fields: this.props.fields,
			formDependency: this.props.formDependency
		});

		this.jsonEditor = new CheckConfigJsonEditor({
			json: this.props.json,
			editorDependency: this.formEditor
		});
	}

	getData() {
		// A small repetition to improve type safety
		if (this.mode === 'json') {
			return {
				mode: this.mode,
				value: this.jsonEditor.getData()
			};
		} else {
			return {
				mode: this.mode,
				value: this.formEditor.getData()
			};
		}
	}
}
