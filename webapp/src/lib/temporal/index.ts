// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import TemporalI18nProvider from './temporal-i18n-provider.svelte';
import { workflowStatuses } from '@forkbombeu/temporal-ui';
export { TemporalI18nProvider, workflowStatuses };

//

export type WorkflowStatusType = NonNullable<(typeof workflowStatuses)[number]>;

export function isWorkflowStatus(status?: string | null | undefined): status is WorkflowStatusType {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	return workflowStatuses.includes(status as any);
}
