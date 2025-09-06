// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SuperForm } from 'sveltekit-superforms';

import { zod } from 'sveltekit-superforms/adapters';
import * as Task from 'true-myth/task';
import { z } from 'zod';

import type { FieldsOptions } from '@/collections-components/form/collectionFormTypes';
import type { CredentialIssuersFormData, CredentialIssuersResponse } from '@/pocketbase/types';

import { setupCollectionForm } from '@/collections-components/form/collectionFormSetup';
import { createForm } from '@/forms';
import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors';

//

// TODO: Inject storage dependency / Backend abstraction for testing

export class CredentialIssuerManager {
	constructor() {
		this.initEffects();
	}

	currentTask = $state<ManagerTask>();
	importedIssuer = $state<CredentialIssuersResponse>();

	status = $derived.by(() => {
		if (!this.importedIssuer) return 'init';
		else if (this.importedIssuer) return 'imported_and_editing';
	});

	tasks = {
		import: (url: string) => {
			return Task.fromPromise(
				pb.send('/credentials_issuers/start-check', {
					method: 'POST',
					body: {
						credentialIssuerUrl: url
					}
				}) as Promise<ImportCredentialIssuerResult>
			)
				.map(({ record }) => {
					this.importedIssuer = record;
				})
				.mapRejected((e) => new ImportError(e));
		},

		discardImport: () => {
			if (!this.importedIssuer) return Task.resolve();
			return Task.fromPromise(
				pb.collection('credential_issuers').delete(this.importedIssuer.id)
			)
				.map(() => {
					this.importedIssuer = undefined;
				})
				.mapRejected((e) => new DiscardImportError(e));
		},

		create: (e: unknown) => {
			return Task.reject(new CreateError(e));
		},

		edit: (e: unknown) => {
			return Task.reject(new EditError(e));
		}
	};

	/* Import */

	importForm = createForm({
		adapter: zod(z.object({ url: z.string() })),
		onSubmit: async ({ form }) => {
			const { url } = form.data;
			await this.runTask(this.tasks.import(url));
		}
	});

	/* Discard Import */

	/* Create */

	private fieldsOptions: Partial<FieldsOptions<'credential_issuers'>> = {
		exclude: ['owner', 'imported', 'url', 'workflow_url', 'published']
	};

	createForm = setupCollectionForm({
		collection: 'credential_issuers',
		fieldsOptions: this.fieldsOptions,
		onError: async (e) => {
			this.currentTask = this.tasks.create(e);
		}
	});

	/* Edit */
	// TODO - Can be merged with createForm

	editForm = $state<SuperForm<CredentialIssuersFormData>>();

	$effect_initEditForm() {
		$effect(() => {
			if (!this.importedIssuer) return;
			this.editForm = setupCollectionForm({
				collection: 'credential_issuers',
				recordId: this.importedIssuer.id,
				initialData: this.importedIssuer,
				fieldsOptions: this.fieldsOptions,
				onError: async (e) => {
					this.currentTask = this.tasks.edit(e);
				}
			});
		});
	}

	/* UI */

	sheet = new Window({
		content: {
			importForm: this.importForm,
			createForm: this.createForm,
			editForm: this.editForm
		},
		beforeClose: (preventClose) => {
			if (this.status === 'imported_and_editing') {
				preventClose();
				this.discardAlert.window.open();
			}
		}
	});

	discardAlert = new Alert({
		window: new Window(),
		onConfirm: async () => {
			await this.runTask(this.tasks.discardImport());
			this.sheet.close();
		}
	});

	/* Utils */

	initEffects() {
		Object.entries(this).forEach(([key, value]) => {
			if (typeof value === 'function' && key.startsWith('$effect_')) {
				value();
			}
		});
	}

	async runTask(task: ManagerTask) {
		this.currentTask = task;
		await this.currentTask;
	}
}

export type ImportCredentialIssuerResult = {
	credentialIssuerUrl: string;
	record: CredentialIssuersResponse;
	operation: 'create' | 'update';
};

type ManagerInstance = InstanceType<typeof CredentialIssuerManager>;
type ManagerTasks = ManagerInstance['tasks'];
type ManagerTask = ReturnType<ManagerTasks[keyof ManagerTasks]>;

/* Errors */

class GenericError extends Error {
	constructor(e: unknown) {
		super(getExceptionMessage(e));
	}
}

class ImportError extends GenericError {}
class DiscardImportError extends GenericError {}
class CreateError extends GenericError {}
class EditError extends GenericError {}

/* Window */

type Content = object;
type BeforeClose = (preventDefault: () => void) => void;

class Window<C extends Content = object> {
	constructor(private readonly init: { content?: C; beforeClose?: BeforeClose } = {}) {}

	isOpen = $state(false);

	open() {
		this.isOpen = true;
	}

	close() {
		let prevent = false;
		const preventDefault = () => {
			prevent = true;
		};
		try {
			this.init?.beforeClose?.(preventDefault);
			if (!prevent) this.isOpen = false;
		} catch (e) {
			console.warn(e);
		}
	}
}

/* Alert */

type AlertAction = () => void | Promise<void>;

class Alert<W extends Window> {
	constructor(
		private readonly init: { window: W; onConfirm?: AlertAction; onDismiss?: AlertAction }
	) {}

	get window() {
		return this.init.window;
	}

	async confirm() {
		await this.init.onConfirm?.();
		this.init.window.close();
	}

	async dismiss() {
		await this.init.onDismiss?.();
		this.init.window.close();
	}
}
