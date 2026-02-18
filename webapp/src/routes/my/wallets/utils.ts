// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WalletActionsCategoryOptions } from '@/pocketbase/types';

import { m } from '@/i18n';

//

export const walletActionCategories: Record<WalletActionsCategoryOptions, string> = {
	onboarding: m.Onboarding(),
	'get-credential-generic': `${m.Get_credential()} (${m.Generic()})`,
	'get-credential-specific': `${m.Get_credential()}`,
	'verify-credential': m.Verify_credential(),
	other: m.Other()
};
