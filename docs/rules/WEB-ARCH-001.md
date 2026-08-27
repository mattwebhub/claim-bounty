# WEB-ARCH-001 — Web dependency direction

Shared modules import no app, route, or feature code. Features import shared code and expose a narrow `index.ts`. Routes compose feature public entrypoints. App modules own providers and routing composition.

Run `pnpm --filter @micro1/web architecture:check` to verify the executable dependency graph.
