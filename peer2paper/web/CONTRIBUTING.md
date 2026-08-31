# Contributing

Thank you for improving Peer2Paper.

1. Create a focused branch from the default branch.
2. Install exactly from the lockfile with `npm ci`.
3. Keep user-facing copy in the locale catalogs and preserve the English fallback schema.
4. Do not commit audit source material, participant data, answer keys, credentials or internal assessment bundles.
5. Run `npm run ci` before opening a pull request.

Pull requests should explain the user impact, include relevant tests and note any change to authentication, database policies, scientific status language or privacy behavior. Changes to legal text, Row Level Security or result interpretation require domain review in addition to code review.
