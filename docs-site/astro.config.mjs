// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
  site: 'https://mr-pmillz.github.io',
  base: '/scrapedoctl',
  integrations: [
    starlight({
      title: 'scrapedoctl',
      description:
        'A professional-grade CLI tool and MCP server that bridges AI agents and the web via the Scrape.do API.',
      logo: { src: './src/assets/logo.svg' },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/mr-pmillz/scrapedoctl',
        },
      ],
      defaultLocale: 'root',
      locales: {
        root: { label: 'English', lang: 'en' },
        ru: { label: 'Русский', lang: 'ru' },
      },
      sidebar: [
        {
          label: 'Documentation',
          translations: { ru: 'Документация' },
          items: [
            {
              label: 'Introduction',
              translations: { ru: 'Введение' },
              slug: 'introduction',
            },
            {
              label: 'Installation & Setup',
              translations: { ru: 'Установка и настройка' },
              slug: 'installation',
            },
            {
              label: 'Usage Guide',
              translations: { ru: 'Руководство пользователя' },
              slug: 'usage',
            },
            {
              label: 'Go SDK',
              translations: { ru: 'Go SDK' },
              slug: 'sdk',
            },
            {
              label: 'Architecture & Design',
              translations: { ru: 'Архитектура и дизайн' },
              slug: 'architecture',
            },
          ],
        },
        {
          label: 'Project',
          translations: { ru: 'Проект' },
          items: [
            {
              label: 'Contributing',
              translations: { ru: 'Участие в разработке' },
              link: 'https://github.com/mr-pmillz/scrapedoctl/blob/main/CONTRIBUTING.md',
              attrs: { target: '_blank' },
            },
            {
              label: 'License',
              translations: { ru: 'Лицензия' },
              link: 'https://github.com/mr-pmillz/scrapedoctl/blob/main/LICENSE',
              attrs: { target: '_blank' },
            },
          ],
        },
      ],
      editLink: {
        baseUrl: 'https://github.com/mr-pmillz/scrapedoctl/edit/main/docs-site/',
      },
      lastUpdated: true,
    }),
  ],
});
