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
} from 'lucide-svelte';

import type { IconComponent } from '@/components/types';
import type { CollectionName } from '@/pocketbase/collections-models';

import { m } from '@/i18n';

//

export type EntityData = {
	slug: string;
	icon: IconComponent;
	labels: {
		singular: string;
		plural: string;
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
			bg: 'bg-[hsl(var(--blue-background))]',
			text: 'text-[hsl(var(--blue-foreground))]',
			border: 'border-[hsl(var(--blue-outline))]'
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
			bg: 'bg-[hsl(var(--green-background))]',
			text: 'text-[hsl(var(--green-foreground))]',
			border: 'border-[hsl(var(--green-outline))]'
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
			bg: 'bg-[hsl(var(--green-light-background))]',
			text: 'text-[hsl(var(--green-light-foreground))]',
			border: 'border-[hsl(var(--green-light-outline))]'
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
			bg: 'bg-[hsl(var(--red-background))]',
			text: 'text-[hsl(var(--red-foreground))]',
			border: 'border-[hsl(var(--red-outline))]'
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
			bg: 'bg-[hsl(var(--orange-background))]',
			text: 'text-[hsl(var(--orange-foreground))]',
			border: 'border-[hsl(var(--orange-outline))]'
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
			bg: 'bg-[hsl(var(--purple-background))]',
			text: 'text-[hsl(var(--purple-foreground))]',
			border: 'border-[hsl(var(--purple-outline))]'
		}
	},

	test_runs: {
		slug: 'test-runs',
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
