// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod/v3';

export const ConformanceCheckSchema = z.object({
	runId: z.string(),
	standard: z.string(),
	test: z.string(),
	workflowId: z.string(),
	status: z.string()
});

export type ConformanceCheck = z.infer<typeof ConformanceCheckSchema>;
