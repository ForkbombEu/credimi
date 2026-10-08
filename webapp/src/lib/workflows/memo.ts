// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod/v3';

import { warn } from '@/utils/other';

//

const memoFieldSchema = z.object({
	data: z.string(),
	metadata: z.object({
		encoding: z.string()
	})
});

type MemoField = z.infer<typeof memoFieldSchema>;

export type WorkflowMemo = {
	author: string;
	standard: string;
	test: string;
	has_qr?: boolean;
};

const credimiCapabilitiesSchema = z.object({
	logs: z.boolean().optional(),
	qr: z.boolean().optional()
});

/** Parse Credimi memo fields from Temporal describe `memo` (protojson). */
export function getWorkflowMemo(memo: unknown): WorkflowMemo | undefined {
	try {
		if (!memo || typeof memo !== 'object' || !('fields' in memo)) {
			return undefined;
		}

		const fields = z.record(memoFieldSchema).parse((memo as { fields: unknown }).fields);
		if (!fields) return undefined;
		const author = memoFieldToText(fields['author']);
		const standard = memoFieldToText(fields['standard']);
		const test = memoFieldToText(fields['test'])?.split('/').at(-1)?.split('.').at(0);
		if (!author || !standard || !test) return undefined;
		return {
			author,
			standard,
			test,
			has_qr: memoFieldHasQR(fields['credimi_capabilities'])
		};
	} catch (error) {
		warn('Failed to parse memo:', error);
		return undefined;
	}
}

function memoFieldToText(field: MemoField | undefined) {
	if (!field) return undefined;
	try {
		const { data } = field;
		return atob(data).replaceAll('"', '').trim();
	} catch (error) {
		throw new Error(`Failed to decode memo field: ${error}`);
	}
}

function memoFieldHasQR(field: MemoField | undefined): boolean {
	if (!field) return false;
	try {
		const parsed = credimiCapabilitiesSchema.safeParse(JSON.parse(atob(field.data)));
		return parsed.success ? Boolean(parsed.data.qr) : false;
	} catch {
		return false;
	}
}
