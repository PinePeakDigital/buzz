# buzz documentation site

The documentation site for [buzz](https://github.com/PinePeakDigital/buzz),
published at <https://buzz.nathanarthur.com>. Built with
[Starlight](https://starlight.astro.build/) (Astro).

## Local development

```bash
npm install
npm run dev      # Dev server at http://localhost:4321
npm run build    # Build the production site into dist/
npm run preview  # Preview the production build locally
```

## Content

Documentation pages live in `src/content/docs/`. The sidebar is configured in
`astro.config.mjs`.

## Deployment

The site is hosted as a Render static site, configured by `render.yaml` at the
repository root. It auto-deploys on every push to `main`, and each pull request
gets its own preview URL.
