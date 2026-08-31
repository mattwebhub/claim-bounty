# Security policy

## Supported versions

Security fixes are applied to the latest version on the default branch.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Contact the security address configured by the Peer2Paper operator and include only the minimum information needed to reproduce the issue. Do not attach research datasets, credentials, access tokens, personal data or participant records to the initial report.

The operator should acknowledge a report within three business days and coordinate disclosure after a fix is available. A production deployment must replace the example contact in `.env.example` with an actively monitored address before launch.

## Deployment responsibilities

This repository never requires a Supabase service-role key in the browser or build environment. Operators are responsible for enabling Row Level Security, applying the checked-in migration, configuring custom SMTP and Google OAuth, reviewing authentication rate limits, and keeping framework and hosting dependencies patched.
