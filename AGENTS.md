# Peer2Paper agent guardrails

## Public-site invariants

- The canonical public app is `peer2paper/web`.
- Preserve the established root-page composition: the brain background, localized paper dropbox, and four-step audit explanation. Do not replace or redesign that content unless the user explicitly asks for a redesign.
- Preserve the shared localized header and footer.
- The canonical brand mark is the fox with a magnifying glass in `peer2paper/web/public/peer2paper-fox-loupe.png`. Use it for the header, footer, upload UI, favicon, social cards, and repository artwork. Do not introduce the former diamond-and-dot placeholder.
- Run `npm --prefix peer2paper/web run ci` after public-site changes.

## Production deployment

- Deploy the public site only with `./scripts/deploy-web-production` from the repository root.
- Do not run `vercel deploy` directly from `peer2paper/web`: the Vercel project already has `peer2paper/web` configured as its Root Directory, so doing that resolves to `peer2paper/web/peer2paper/web`.
- Do not run a raw root deployment either: it uploads unrelated monorepo and private working material.
- The deployment wrapper builds a minimal temporary root with only the public web package, runs the complete web CI gate, and invokes the linked Vercel project from the expected root layout.

