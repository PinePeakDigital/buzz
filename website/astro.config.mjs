// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://buzz.nathanarthur.com',
	integrations: [
		starlight({
			title: 'buzz',
			description:
				'A terminal user interface and CLI for Beeminder, built with Bubble Tea.',
			components: {
				Footer: './src/components/Footer.astro',
			},
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/PinePeakDigital/buzz',
				},
			],
			editLink: {
				baseUrl: 'https://github.com/PinePeakDigital/buzz/edit/main/website/',
			},
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'getting-started/introduction' },
						{ label: 'Installation', slug: 'getting-started/installation' },
						{ label: 'Authentication', slug: 'getting-started/authentication' },
						{ label: 'Configuration', slug: 'getting-started/configuration' },
					],
				},
				{
					label: 'Commands',
					items: [
						{ label: 'Overview', slug: 'commands/overview' },
						{ label: 'Viewing goals', slug: 'commands/viewing' },
						{ label: 'Managing goals', slug: 'commands/managing' },
					],
				},
				{
					label: 'Interactive TUI',
					items: [{ label: 'Using the TUI', slug: 'guides/tui' }],
				},
				{
					label: 'Contributing',
					items: [{ label: 'Development', slug: 'guides/development' }],
				},
			],
		}),
	],
});
