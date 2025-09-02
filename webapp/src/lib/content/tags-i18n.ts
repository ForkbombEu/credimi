// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { m } from '@/i18n';

import tagslist from './tags-list.generated.json';

export type Tag = keyof typeof tagslist;

const DEFAULT_UNKNOWN_LABEL = m.tag_unknown_tag();

export const tagsTranslations: Record<Tag, string> = {
	capabilities: m.tag_capabilities(),
	'conformance-and-compliance-tools': m.tag_conformance_and_compliance_tools(),
	onboarding: m.tag_onboarding(),
	roadmap: m.tag_roadmap(),
	performance: m.tag_performance(),
	monitoring: m.tag_monitoring(),
	platform: m.tag_platform(),
	'conformance-automation': m.tag_conformance_automation(),
	sdk: m.tag_sdk(),
	governance: m.tag_governance(),
	'release-notes': m.tag_release_notes(),
	migration: m.tag_migration(),
	solutions: m.tag_solutions(),
	'financial-services-and-kyc': m.tag_financial_services_and_kyc(),
	tutorial: m.tag_tutorial(),
	testing: m.tag_testing(),
	developers: m.tag_developers(),
	'sdks-and-libraries': m.tag_sdks_and_libraries(),
	overview: m.tag_overview(),
	api: m.tag_api(),
	guide: m.tag_guide(),
	company: m.tag_company(),
	'about-credimi': m.tag_about_credimi(),
	automation: m.tag_automation(),
	architecture: m.tag_architecture(),
	'credential-issuance': m.tag_credential_issuance(),
	community: m.tag_community(),
	optimization: m.tag_optimization(),
	analytics: m.tag_analytics(),
	resources: m.tag_resources(),
	documentation: m.tag_documentation(),
	configuration: m.tag_configuration(),
	security: m.tag_security(),
	setup: m.tag_setup(),
	compliance: m.tag_compliance(),
	'press-releases': m.tag_press_releases(),
	deployment: m.tag_deployment(),
	'transportation-and-smart-mobility': m.tag_transportation_and_smart_mobility(),
	troubleshooting: m.tag_troubleshooting(),
	'ecosystem-governance': m.tag_ecosystem_governance(),
	faq: m.tag_faq(),
	changelog: m.tag_changelog(),
	'credential-verification': m.tag_credential_verification(),
	'api-documentation': m.tag_api_documentation(),
	reference: m.tag_reference(),
	'digital-identity-for-governments': m.tag_digital_identity_for_governments(),
	blog: m.tag_blog(),
	'best-practices': m.tag_best_practices(),
	careers: m.tag_careers(),
	scaling: m.tag_scaling(),
	'case-studies': m.tag_case_studies(),
	'deployment-options-cloud-on-prem': m.tag_deployment_options_cloud_on_prem(),
	support: m.tag_support(),
	github: m.tag_github(),
	'test-suites-and-runs': m.tag_test_suites_and_runs(),
	videos: m.tag_videos(),
	'get-started-with-the-api': m.tag_get_started_with_the_api(),
	'articles-and-tutorials': m.tag_articles_and_tutorials(),
	integration: m.tag_integration(),
	'workforce-verification': m.tag_workforce_verification(),
	'contact-us': m.tag_contact_us(),
	'sandbox-environment': m.tag_sandbox_environment(),
	'compliance-and-standards': m.tag_compliance_and_standards(),
	'identity-wallet-integration': m.tag_identity_wallet_integration(),
	'legal-and-privacy': m.tag_legal_and_privacy(),
	'events-and-webinars': m.tag_events_and_webinars(),
	'developer-tools': m.tag_developer_tools(),
	'education-credentials': m.tag_education_credentials(),
	'partner-network': m.tag_partner_network(),
	'join-our-slack': m.tag_join_our_slack(),
	'our-vision-and-approach': m.tag_our_vision_and_approach(),
	'platform-architecture': m.tag_platform_architecture(),
	'conformance-community-forum': m.tag_conformance_community_forum()
};

export function getTagTranslation(
	tag: string,
	defaultLabel: string = DEFAULT_UNKNOWN_LABEL
): string {
	if ((tag as Tag) in tagsTranslations) {
		return tagsTranslations[tag as Tag];
	}
	return defaultLabel;
}
