# scrapedoctl docs site

Astro + [Starlight](https://starlight.astro.build/) source for the public documentation hosted at <https://mr-pmillz.github.io/scrapedoctl/>.

## Local development

```bash
cd docs-site
npm install
npm run dev      # http://localhost:4321/scrapedoctl
npm run build    # static output in dist/
npm run preview  # serve dist/ locally
```

## Authoring

Content lives in `src/content/docs/`. Each page is a Markdown or MDX file with Starlight frontmatter (`title`, optional `description`, optional `sidebar`). The English docs are at the root of the collection; the Russian translations live under `ru/`. Sidebar ordering and labels are configured centrally in `astro.config.mjs`.

## Deployment

The `Deploy docs` workflow in `.github/workflows/deploy-docs.yml` rebuilds and publishes the site to GitHub Pages on every push to `main` that touches `docs-site/`.
