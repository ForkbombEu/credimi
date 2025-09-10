// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import z from 'zod';

//

export function zodFileSchema(options: { mimeTypes?: string[]; maxSize?: number } = {}) {
	const { mimeTypes, maxSize } = options;
	let s = z.instanceof(File);
	if (mimeTypes && mimeTypes.length > 0) {
		const mimes = mimeTypes as readonly string[];
		s = s.refine((file) => mimes.includes(file.type), `File type not: ${mimes.join(', ')}`);
	}
	if (maxSize) {
		s.refine((file) => file.size < maxSize, `File size bigger than ${maxSize} bytes`);
	}
	return s;
}

//

export function readFileAsDataURL(file: File): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.readAsDataURL(file);
		reader.onload = () => {
			resolve(reader.result as string);
		};
		reader.onerror = () => {
			reject(reader.error);
		};
	});
}

export async function readFileAsBase64(file: File): Promise<string> {
	const dataURL = await readFileAsDataURL(file);
	return dataURL.split(',')[1];
}

export function readFileAsString(file: File): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.readAsText(file);
		reader.onload = () => {
			const result = (reader.result as string).trim();
			resolve(result);
		};
		reader.onerror = () => {
			reject(reader.error);
		};
	});
}

export function startFileUpload<Multiple extends boolean = false>(
	options: {
		accept?: string;
		multiple?: Multiple;
		onLoad?: (file: Multiple extends true ? File[] : File) => void | Promise<void>;
	} = {}
) {
	const { accept, multiple = false, onLoad } = options;

	const input = document.createElement('input');
	input.type = 'file';
	input.multiple = multiple;
	if (accept) input.accept = accept;

	input.onchange = async (e) => {
		if (!(e.target instanceof HTMLInputElement)) return;
		if (!e.target.files) return;
		if (multiple) {
			const files = Array.from(e.target.files);
			// @ts-expect-error TODO: fix this
			await onLoad?.(files);
		} else {
			const file = e.target.files[0];
			// @ts-expect-error TODO: fix this
			await onLoad?.(file);
		}
	};

	input.click();
}
