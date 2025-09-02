// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { CheckConfigCodeEditor } from '$lib/start-checks-form/configure-checks-form/check-config-code-editor';
import {
	DependentCheckConfigFormEditor,
	type DependentCheckConfigFormEditorProps
} from '$lib/start-checks-form/configure-checks-form/check-config-form-editor';

import type { BaseEditor } from '../../_utils';

//

type CheckConfigEditorProps = DependentCheckConfigFormEditorProps & {
	id: string;
	code: string;
};

export type CheckConfigEditorMode = 'code' | 'form';

export class CheckConfigEditor implements BaseEditor {
	public readonly codeEditor: CheckConfigCodeEditor;
	public readonly formEditor: DependentCheckConfigFormEditor;

	mode: CheckConfigEditorMode = $derived.by(() => (this.codeEditor.isTainted ? 'code' : 'form'));

	isValid = $derived.by(() =>
		this.mode === 'code' ? this.codeEditor.isValid : this.formEditor.isValid
	);

	constructor(public readonly props: CheckConfigEditorProps) {
		this.formEditor = new DependentCheckConfigFormEditor({
			fields: this.props.fields,
			formDependency: this.props.formDependency
		});

		this.codeEditor = new CheckConfigCodeEditor({
			code: this.props.code,
			editorDependency: this.formEditor
		});
	}

	getData() {
		// A small repetition to improve type safety
		if (this.mode === 'code') {
			return {
				mode: this.mode,
				value: this.codeEditor.getData()
			};
		} else {
			return {
				mode: this.mode,
				value: this.formEditor.getData()
			};
		}
	}
}
