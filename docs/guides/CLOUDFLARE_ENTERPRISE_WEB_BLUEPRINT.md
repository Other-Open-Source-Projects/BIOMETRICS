# Cloudflare Enterprise Web Blueprint (Free-First)

Last updated: 2026-02-26

## Objective

Deliver a website that looks premium, performs globally, and keeps operating/security costs at or near zero.

## Stack decision (implemented)

1. Next.js/Nextra as source framework.
2. Cloudflare Pages as global edge host.
3. Static export (`out/`) for deterministic deploy artifacts and low operational risk.
4. Security headers enforced for static hosting via `website/public/_headers`.

## Why this is enterprise-grade

1. Global edge delivery by default (Cloudflare network).
2. Deterministic immutable artifacts for release audits.
3. Environment-independent deploy command (`pnpm run deploy:cloudflare`).
4. Security baseline in source control (CSP, HSTS, COOP/CORP, Permissions-Policy).

## Free-first economics

1. Pages Free supports production hosting with preview flow.
2. Cloudflare Web Analytics is available as a free product tier.
3. Cloudflare Turnstile offers free bot/challenge protection for forms.

## Release operating model

1. `main` deploys production.
2. PR branches deploy preview.
3. CI must pass `pnpm run build`, `pnpm run build:cloudflare`, content checks, and e2e gates.
4. Monthly visual and KPI review: bounce, conversion, Core Web Vitals, accessibility.

## Commands

```bash
cd website
pnpm run cf:project:create   # first-time only
pnpm run deploy:cloudflare
```

## Sources

- Cloudflare Pages overview (features + limits): <https://developers.cloudflare.com/pages/platform/>
- Cloudflare Next.js on Workers (framework support): <https://developers.cloudflare.com/workers/framework-guides/web-apps/nextjs/>
- Wrangler Pages deploy command: <https://developers.cloudflare.com/workers/wrangler/commands/#pages-deploy>
- Wrangler authentication (`wrangler whoami`): <https://developers.cloudflare.com/workers/wrangler/commands/#whoami>
- Cloudflare Web Analytics: <https://developers.cloudflare.com/analytics/web-analytics/>
- Cloudflare Turnstile: <https://developers.cloudflare.com/turnstile/>
- Core Web Vitals guidance: <https://web.dev/articles/vitals>
- WCAG 2.2 recommendation: <https://www.w3.org/TR/WCAG22/>
