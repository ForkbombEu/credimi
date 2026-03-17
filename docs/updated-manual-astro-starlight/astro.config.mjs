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
          autogenerate: { directory: 'manual' },
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