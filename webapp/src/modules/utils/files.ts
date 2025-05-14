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
