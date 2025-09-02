// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import CheckConfigFormEditorComponent from './check-config-form-editor.svelte';
import { CheckConfigFormEditor } from './check-config-form-editor.svelte.js';
import DependentCheckConfigFormEditorComponent from './dependent-check-config-form-editor.svelte';
import {
	DependentCheckConfigFormEditor,
	type DependentCheckConfigFormEditorProps
} from './dependent-check-config-form-editor.svelte.js';

export {
	CheckConfigFormEditor as CheckConfigFormEditor,
	CheckConfigFormEditorComponent,
	DependentCheckConfigFormEditor as DependentCheckConfigFormEditor,
	DependentCheckConfigFormEditorComponent,
	type DependentCheckConfigFormEditorProps
};
