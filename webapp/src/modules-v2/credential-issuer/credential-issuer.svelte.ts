// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { form, pocketbase as pb, pocketbaseCrud, types as t, task, ui } from '#';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod';

import type { CredentialIssuersResponse } from '@/pocketbase/types';

//

// TODO: Inject storage dependency / Backend abstraction for testing

type Crud = pocketbaseCrud.Instance<'credential_issuers'>;
type ImportForm = form.Instance<{ url: string }>;
type RecordForm = pb.recordform.Instance<'credential_issuers'>;

type ManagerDependencies = {
	crud: Crud;
	importCredentialIssuer: ImportCredentialIssuer;
};

export class Manager extends task.Runner implements ManagerDependencies {
	// Dependencies
	crud: Crud;
	importCredentialIssuer: ImportCredentialIssuer;

	// Components
	recordForm: RecordForm;
	importForm: ImportForm;
	sheet: ui.Window<{ importForm: ImportForm; recordForm: RecordForm }>;
	discardAlert: ui.Alert;

	// State
	importedIssuer = $state<CredentialIssuersResponse>();
	currentState: 'init' | 'update' | 'imported_and_editing' = $derived.by(() => {
		if (this.importedIssuer && this.recordForm.currentMode === 'update') {
			return 'imported_and_editing';
		} else if (this.recordForm.currentMode === 'update') {
			return 'update';
		} else {
			return 'init';
		}
	});

	constructor(init: Partial<ManagerDependencies> = {}) {
		super();

		this.crud = init.crud ?? new pocketbaseCrud.Instance('credential_issuers');
		this.importCredentialIssuer = init.importCredentialIssuer ?? importCredentialIssuer;

		this.recordForm = new pb.recordform.Instance<'credential_issuers'>({
			collection: 'credential_issuers',
			mode: 'create',
			initialData: {},
			crud: this.crud,
			exclude: ['owner', 'imported', 'workflow_url', 'published', { update: ['url'] }]
		});

		this.importForm = new form.Instance({
			adapter: zod(z.object({ url: z.string() })),
			onSubmit: async ({ url }) => {
				await this.run(this.import(url));
			}
		});

		this.sheet = new ui.Window({
			content: {
				importForm: this.importForm,
				recordForm: this.recordForm
			},
			beforeClose: (preventClose) => {
				if (this.currentState === 'imported_and_editing') {
					preventClose();
					this.discardAlert.window.open();
				}
			}
		});

		this.discardAlert = new ui.Alert({
			window: new ui.Window(),
			onConfirm: async () => {
				await this.run(this.discardImport());
				this.sheet.close();
			}
		});

		$effect(() => {
			if (!this.importedIssuer) return;
			this.recordForm.setMode({
				mode: 'update',
				record: this.importedIssuer
			});
		});
	}

	/* Import */

	import(url: string) {
		return task.withError(this.importCredentialIssuer(url), ImportError).map(({ record }) => {
			this.importedIssuer = record;
		});
	}

	discardImport() {
		if (!this.importedIssuer) return task.resolve();
		return this.crud
			.delete(this.importedIssuer.id)
			.map(() => {
				this.importedIssuer = undefined;
			})
			.mapRejected((e) => new DiscardImportError(e));
	}
}

/* Import */

type ImportCredentialIssuer = (url: string) => Promise<ImportCredentialIssuerResult>;

export const importCredentialIssuer: ImportCredentialIssuer = (url) => {
	return pb.defaultClient.send('/credentials_issuers/start-check', {
		method: 'POST',
		body: {
			credentialIssuerUrl: url
		}
	});
};

export const testImportCredentialIssuer: (
	record: CredentialIssuersResponse
) => ImportCredentialIssuer = (record) => async (url) => ({
	credentialIssuerUrl: url,
	operation: 'create',
	record
});

type ImportCredentialIssuerResult = {
	credentialIssuerUrl: string;
	record: CredentialIssuersResponse;
	operation: 'create' | 'update';
};

/* Types */

class ImportError extends t.BaseError {}
class DiscardImportError extends t.BaseError {}
