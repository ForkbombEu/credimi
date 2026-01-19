// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	CheckCheck,
	CheckCircle,
	QrCode,
	SheetIcon,
	ShieldCheck,
	TestTube2,
	Users,
	Wallet,
	WaypointsIcon
} from '@lucide/svelte';

import type { IconComponent } from '@/components/types';
import type { CollectionName } from '@/pocketbase/collections-models';

import { m } from '@/i18n';

//

export type EntityData = {
	slug: string;
	icon: IconComponent;
	labels: {
		singular: string;
		plural?: string;
	};
	classes: {
		bg: string;
		text: string;
		border: string;
	};
};

type EntitiesConfig = Partial<
	Record<CollectionName | 'conformance_checks' | 'test_runs', EntityData>
>;

//

export const entities = {
	wallets: {
		slug: 'wallets',
		icon: Wallet,
		labels: {
			singular: m.Wallet(),
			plural: m.Wallets()
		},
		classes: {
			bg: 'bg-blue-100',
			text: 'text-blue-600',
			border: 'border-blue-600'
		}
	},

	credential_issuers: {
		slug: 'credential-issuers',
		icon: Users,
		labels: {
			singular: m.Issuer(),
			plural: m.Issuers()
		},
		classes: {
			bg: 'bg-green-100',
			text: 'text-green-600',
			border: 'border-green-600'
		}
	},

	credentials: {
		slug: 'credentials',
		icon: QrCode,
		labels: {
			singular: m.Credential(),
			plural: m.Credentials()
		},
		classes: {
			bg: 'bg-lime-100',
			text: 'text-lime-600',
			border: 'border-lime-600'
		}
	},

	verifiers: {
		slug: 'verifiers',
		icon: ShieldCheck,
		labels: {
			singular: m.Verifier(),
			plural: m.Verifiers()
		},
		classes: {
			bg: 'bg-red-100',
			text: 'text-red-600',
			border: 'border-red-600'
		}
	},

	use_cases_verifications: {
		slug: 'use-case-verifications',
		icon: CheckCircle,
		labels: {
			singular: m.Use_case_verification(),
			plural: m.Use_case_verifications()
		},
		classes: {
			bg: 'bg-orange-100',
			text: 'text-orange-600',
			border: 'border-orange-600'
		}
	},

	pipelines: {
		slug: 'pipelines',
		icon: WaypointsIcon,
		labels: {
			singular: m.Pipeline(),
			plural: m.Pipelines()
		},
		classes: {
			bg: 'bg-orange-100',
			text: 'text-orange-600',
			border: 'border-orange-600'
		}
	},

	conformance_checks: {
		slug: 'conformance-checks',
		icon: SheetIcon,
		labels: {
			singular: m.Conformance_check(),
			plural: m.Conformance_Checks()
		},
		classes: {
			bg: 'bg-red-100',
			text: 'text-red-500',
			border: 'border-red-500'
		}
	},

	custom_checks: {
		slug: 'custom-checks',
		icon: CheckCheck,
		labels: {
			singular: m.Custom_check(),
			plural: m.Custom_checks()
		},
		classes: {
			bg: 'bg-purple-100',
			text: 'text-purple-600',
			border: 'border-purple-600'
		}
	},

	test_runs: {
		slug: 'tests/runs',
		icon: TestTube2,
		labels: {
			singular: m.Test_run(),
			plural: m.Test_runs()
		},
		classes: {
			bg: 'bg-black/10',
			text: 'text-black',
			border: 'border-black'
		}
	}
} satisfies EntitiesConfig;
