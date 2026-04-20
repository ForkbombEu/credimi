import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import credimiLogo from './src/content/docs/images/logo/credimi_logo-transp_emblem.png';

export default defineConfig({
  integrations: [
    starlight({
      title: 'Credimi Docs',
      logo: { src: credimiLogo },
      sidebar: [
        {
          label: 'Manual',
          items: [
            { label: 'Overview', link: '/manual/' },
            {
              label: 'Marketplace',
              items: [
                { label: 'Overview', link: '/manual/marketplace/' },
                { label: 'Explore Components and Solutions', link: '/manual/marketplace/explore-components-and-solutions/' },
                { label: 'Try Issuance and Verification Services', link: '/manual/marketplace/try-issuance-and-verification-services/' },
              ],
            },
            {
              label: 'Conformance & Interop',
              items: [
                { label: 'Overview', link: '/manual/conformance/' },
                { label: 'Run Conformance Checks', link: '/manual/conformance/run-conformance-checks/' },
                { label: 'Try Interop Flows', link: '/manual/conformance/try-interop-flows/' },
              ],
            },
            {
              label: 'Publish to Marketplace',
              items: [
                { label: 'Overview', link: '/manual/publish-to-marketplace/' },
                { label: 'Create an Account and Organization', link: '/manual/publish-to-marketplace/create-account-and-organization/' },
                { label: 'List Wallets, Issuers and Verifiers', link: '/manual/publish-to-marketplace/list-wallets-issuers-and-verifiers/' },
                { label: 'Integrate Issuers and Verifiers with StepCI', link: '/manual/publish-to-marketplace/integrate-with-stepci/' },
				{ label: 'StepCI and Maestro scripting examples', link: '/manual/publish-to-marketplace/yaml-examples/' },
              ],
            },
            {
              label: 'Testing Automation',
              items: [
                { label: 'Overview', link: '/manual/testing-automation/' },
                { label: 'Create Maestro Actions', link: '/manual/testing-automation/create-maestro-actions/' },
                { label: 'Build and Run Pipelines', link: '/manual/testing-automation/build-and-run-pipelines/' },
                { label: 'Inspect Pipeline Execution', link: '/manual/testing-automation/inspect-pipeline-execution/' },
				{ label: 'StepCI and Maestro scripting examples', link: '/manual/publish-to-marketplace/yaml-examples/' },
              ],
            },
          ],
        },
        {
          label: 'Software Architecture',
          autogenerate: { directory: 'software-architecture' },
        },
        {
          label: 'Legal',
          autogenerate: { directory: 'legal' },
        },
      ],
    }),
  ],
});
