// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SuperForm } from 'sveltekit-superforms';

import { pipe, Tuple } from 'effect';

import type { FileManagerValidator, RejectedFile } from '@/components/ui-custom/fileManager.svelte';
import type { GenericRecord, If, IsArray } from '@/utils/types';

import type { FORM_ERROR_PATH } from '../form';

/* Files validation
 *
 * - Objective
 * Here we are trying to validate files *before* they arrive into the form
 *
 * - Reason(s)
 * If a single file is bad, it must not replace a good one
 * If multiple files are uploaded, good ones must be saved and bad ones discarded
 *
 * - Restated objective
 * We want to reject bad files and notify the user about it
 */

export function createFilesValidator<F extends SuperForm<GenericRecord>>(
	form: F,
	field: string,
	multiple: boolean
): FileManagerValidator {
	/**
	 *  - Implementation
	 * To validate the file field, *before* actually adding the files in the form
	 * we can use the `validate` function from superforms, but in "preview mode"
	 * and use it as validator for `FileManager` component
	 */

	const { validate } = form;

	function previewValidate<T = unknown>(data: T) {
		// TODO: this should throw when "field" is not valid
		return validate(field, {
			value: data,
			taint: false,
			update: false
		}) as Promise<ValidationResult<T>>;
	}

	/* Type fix
	 *
	 * Superforms `validate` function return type is not correct:
	 * - Function signature says it is `string[] | undefined`
	 * - If you validate an array value, it is actually `Record<number, string[]> | undefined`
	 */

	type ValidationResult<Input> =
		| undefined
		| If<IsArray<Input>, Record<number | typeof FORM_ERROR_PATH, string[]>, string[]>;

	/* Files validator */

	const validator: FileManagerValidator = async (newFiles) => {
		if (!multiple) {
			const newFile = newFiles[0];
			const validationResult = await previewValidate(newFile);
			const acceptedFiles: File[] = [];
			const rejectedFiles: RejectedFile[] = [];
			if (!validationResult) acceptedFiles.push(newFile);
			else
				rejectedFiles.push({
					file: newFile,
					reasons: validationResult
				});
			return {
				acceptedFiles,
				rejectedFiles
			};
		} else {
			/**
			 * - Issue
			 * We cannot validate multiple files together with superforms [validate] function
			 * - Reason
			 * The validation result is a record where the keys are not linked to the files
			 * - To reproduce, run here:
			 * `console.log(await previewFieldValidation(name, newfiles))`
			 * and upload multiple files, some of which invalid
			 *
			 * - Solution
			 * We validate each file as if it is an array, then extract the result
			 * `previewFieldValidation(name, [f])`
			 */
			const validationEntries = await pipe(
				newFiles.map((f) =>
					previewValidate([f])
						.then((result) => (result ? result[0] : []))
						.then((result) => Tuple.make(f, result))
				),
				(entries) => Promise.all(entries)
			);

			const acceptedFiles: File[] = validationEntries
				.filter(([, errors]) => errors.length == 0)
				.map(([file]) => file);

			const rejectedFiles: RejectedFile[] = validationEntries
				.filter(([, errors]) => errors.length > 0)
				.map(([file, errors]) => ({
					file,
					reasons: errors
				}));

			return {
				acceptedFiles,
				rejectedFiles
			};
		}
	};

	return validator;
}
