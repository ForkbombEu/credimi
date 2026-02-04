// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Memo } from '@forkbombeu/temporal-ui/dist/types';

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
};

export function getWorkflowMemo(workflow: { memo: Memo }): WorkflowMemo | undefined {
	try {
		if (!workflow.memo || !workflow.memo['fields']) {
			return undefined;
		}

		const fields = z.record(memoFieldSchema).parse(workflow.memo['fields']);
		if (!fields) return undefined;
		const author = memoFieldToText(fields['author']);
		const standard = memoFieldToText(fields['standard']);
		const test = memoFieldToText(fields['test'])?.split('/').at(-1)?.split('.').at(0);
		if (!author || !standard || !test) return undefined;
		return {
			author,
			standard,
			test
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
