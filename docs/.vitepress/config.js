import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'StackRun',
  description: 'Self-hosted PaaS deploy platform',
  ignoreDeadLinks: true,
  
  locales: {
    root: {
      label: 'Português',
      lang: 'pt-BR',
      themeConfig: {
        nav: [
          { text: 'Início', link: '/' },
          { text: 'Guias', link: '/guides/' },
          { text: 'API', link: '/api/' },
        ],
        sidebar: {
          '/': [
            {
              text: 'Introdução',
              items: [
                { text: 'O que é', link: '/' },
                { text: 'Como rodar', link: '/guides/getting-started' },
              ]
            },
            {
              text: 'Guias',
              items: [
                { text: 'Deploy via Git', link: '/guides/git-deploy' },
                { text: 'CLI', link: '/guides/cli' },
                { text: 'Docker', link: '/guides/docker' },
                { text: 'Docker Compose', link: '/guides/compose' },
                { text: 'Domínios', link: '/guides/domains' },
                { text: 'Volumes', link: '/guides/volumes' },
                { text: 'SDKs', link: '/guides/sdk' },
                { text: 'CI/CD', link: '/guides/ci-cd' },
              ]
            },
            {
              text: 'Referência',
              items: [
                { text: 'API REST', link: '/api/rest' },
                { text: 'Variáveis de Ambiente', link: '/api/env' },
              ]
            }
          ]
        }
      }
    },
    en: {
      label: 'English',
      lang: 'en-US',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/en/' },
          { text: 'Guides', link: '/en/guides/' },
          { text: 'API', link: '/en/api/' },
        ],
        sidebar: {
          '/en/': [
            {
              text: 'Introduction',
              items: [
                { text: 'What is it', link: '/en/' },
                { text: 'Getting Started', link: '/en/guides/getting-started' },
              ]
            },
            {
              text: 'Guides',
              items: [
                { text: 'Git Deploy', link: '/en/guides/git-deploy' },
                { text: 'CLI', link: '/en/guides/cli' },
                { text: 'Docker', link: '/en/guides/docker' },
                { text: 'Docker Compose', link: '/en/guides/compose' },
                { text: 'Custom Domains', link: '/en/guides/domains' },
                { text: 'Volumes', link: '/en/guides/volumes' },
                { text: 'SDKs', link: '/en/guides/sdk' },
                { text: 'CI/CD', link: '/en/guides/ci-cd' },
                { text: 'Quick Start', link: '/en/guides/quick-start' },
              ]
            },
            {
              text: 'Reference',
              items: [
                { text: 'REST API', link: '/en/api/rest' },
                { text: 'Environment Variables', link: '/en/api/env' },
              ]
            }
          ]
        }
      }
    }
  },
  
  themeConfig: {
    socialLinks: [
      { icon: 'github', link: 'https://github.com/mateussiqueira/stackrun' }
    ]
  }
})
