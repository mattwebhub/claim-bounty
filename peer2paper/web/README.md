# Peer2Paper

The public site and authenticated audit-intake workspace for independent scientific claim audits. It is a Next.js App Router application with eight locale routes, Supabase SSR authentication, Google OAuth and user-owned audit requests protected by Row Level Security.

This package is part of the Peer2Paper monorepo. Scientific source material, private audit runs and grader keys are deliberately excluded from Git. The public sample presents the reviewed aggregate record in [`../examples/elite-party-cues`](../examples/elite-party-cues/README.md); it does not redistribute or grant rights to the underlying research package.

## Local development

Requirements: Node.js 22+ and a Supabase project.

```bash
npm install
cp .env.example .env.local
npm run dev
```

Set `NEXT_PUBLIC_SITE_URL=http://localhost:3000` in `.env.local` when testing authentication locally. The public site builds without Supabase values. Authentication and the workspace deliberately show a configuration state until the environment is connected.

## Production authentication

1. Create a Supabase project and apply [`supabase/migrations/202608310001_create_audit_requests.sql`](supabase/migrations/202608310001_create_audit_requests.sql).
2. Put the project URL and publishable/anon key in `.env.local`. Never expose the service-role key.
3. In Supabase Authentication → URL Configuration, set `https://peer2paper.com` as the production site URL and allow exactly `https://peer2paper.com/auth/callback` plus `http://localhost:3000/auth/callback` for local development. Do not use wildcard redirect origins.
4. In Google Cloud, create an OAuth 2.0 Web client. Add the Supabase callback shown by the Google provider screen (normally `https://<project-ref>.supabase.co/auth/v1/callback`) as an authorized redirect URI.
5. Enable Google under Supabase Authentication → Providers and enter the Google client ID and secret.
6. Configure an SMTP provider and branded email templates before accepting production users. Supabase's default mailer is only suitable for initial testing.
7. Set `NEXT_PUBLIC_SITE_URL=https://peer2paper.com` and `NEXT_PUBLIC_LEGAL_EMAIL` to an actively monitored address.

Email/password sessions and OAuth PKCE codes are exchanged server-side. Authentication callbacks always use the configured canonical origin, and post-authentication destinations are restricted to normalized local paths. The proxy refreshes secure session cookies and guards locale-prefixed `/dashboard` and `/submit` routes. Database RLS remains the authorization boundary even if a route check is bypassed.

## Internationalization

Supported locales match the Toone reference: English, Portuguese, Spanish, French, German, Italian, Dutch and Russian. English is the canonical message schema; locale catalogs override translated product copy and fall back to English for any newly introduced key until localization catches up.

Run the complete local CI sequence before deployment:

```bash
npm run ci
```

This performs type checking, linting, unit and locale-contract tests, a production build, runtime route/security smoke checks and the public-release content scan. The sitemap includes all public locale routes; account and submission routes are excluded from indexing.

## Release checklist

- Replace the example legal email and have privacy/terms reviewed for the operating legal entity and jurisdiction.
- Configure Supabase custom SMTP, CAPTCHA and appropriate Auth rate limits.
- Confirm RLS policies with separate test users; never add a general client update policy for audit status.
- Set hosting secrets, canonical URL, DNS, TLS, CSP/monitoring strategy and error reporting.
- Verify every language route on mobile and desktop and complete professional review of localized scientific terminology.
- Keep restricted source bundles out of `public/`; only the reviewed, authored aggregate case-study record may be published.
