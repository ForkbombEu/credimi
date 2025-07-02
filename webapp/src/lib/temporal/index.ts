// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import TemporalI18nProvider from './temporal-i18n-provider.svelte';
import { workflowStatuses } from '@forkbombeu/temporal-ui';

export type WorkflowStatusType = (typeof workflowStatuses)[number];

export { TemporalI18nProvider, workflowStatuses };
