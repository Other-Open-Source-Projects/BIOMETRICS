# BIOMETRICS Public Website

Public product + documentation website for BIOMETRICS (EN primary, DE secondary).

## Stack

- Next.js + Nextra docs theme
- MDX content in `pages/`
- Playwright E2E + Lighthouse CI + content validators

## Local development

```bash
pnpm install --frozen-lockfile
pnpm run dev
```

## Build + checks

```bash
pnpm run build
pnpm run test:content
pnpm run test:e2e
pnpm run test:lighthouse
```

## Deploy on Cloudflare Pages

First-time project bootstrap:

```bash
pnpm run cf:project:create
```

Production deploy:

```bash
pnpm run deploy:cloudflare
```

Notes:
- Cloudflare build output is static export (`out/`) via `pnpm run build:cloudflare`.
- Security headers for static Pages deploy are shipped via `public/_headers`.
