# ClaimBounty Setup

> **Status**: In-Development | **Updated**: 2026-08-31 | **Scope**: Public web preview, hosted application, and local scientific workflow

## Prerequisites

| Path                | Required software                                           | What it runs                                               |
| ------------------- | ----------------------------------------------------------- | ---------------------------------------------------------- |
| Visual preview      | Node version from `.node-version`, pnpm from `package.json` | Static web interface only                                  |
| Hosted application  | Git, Docker Engine, Docker Compose v2, free local ports     | Web, API, database, storage, scanner, mail sandbox, worker |
| Scientific workflow | macOS, local Toone, Codex or Claude Code, case runtimes     | Parent scientific workflow and its six child routines      |
| Contributor checks  | Hosted prerequisites plus Go version from `apps/api/go.mod` | Linters, contracts, tests, and system gates                |

## Visual-only preview

```sh
pnpm install --frozen-lockfile
pnpm --filter @micro1/web build
pnpm --filter @micro1/web preview
```

This path is useful for interface review. Actions that require the API will not complete.

## Hosted application

1. Copy `.env.example` to a local `.env` file.
2. Replace every demonstration credential before exposing the stack beyond the local machine. Never commit `.env`.
3. Start the ClaimBounty profile:

   ```sh
   docker compose --project-name claimbounty-public --file infra/compose.yaml \
     --profile claimbounty up --build -d --wait
   ```

4. Open the web application at `http://127.0.0.1:8081` and the local mail sandbox at `http://127.0.0.1:8025`. Use the literal `127.0.0.1` host because the API enforces an exact browser-origin match.
5. Stop the stack without deleting its volumes:

   ```sh
   docker compose --project-name claimbounty-public --file infra/compose.yaml \
     --profile claimbounty down
   ```

The hosted path accepts and processes private inputs. Use synthetic data for public demonstrations. Production deployment requires production secrets, TLS, external identity controls, approved storage lifecycle rules, monitoring, backups, and a reviewed data-processing policy.

The hosted boundary ends when an administrator downloads an authorized export with its `Content-Digest`. Scientific execution happens locally through the separately packaged workflow.

## Local scientific workflow

The local scientific workflow requires macOS and the latest [Toone release](https://trytoone.com). Install one supported coding-agent client as well:

- Codex: follow the [official Codex CLI guide](https://learn.chatgpt.com/docs/codex/cli). On macOS or Linux, install it with `curl -fsSL https://chatgpt.com/codex/install.sh | sh`, run `codex`, and sign in with ChatGPT.
- Claude Code: follow the [official Anthropic setup page](https://code.claude.com/docs/en/getting-started). This repository does not duplicate installation commands that may change upstream.

The intended Toone path is:

1. Open Toone and create a project by selecting an empty directory.
2. In that directory, create `claimbounty/input/case-bundle/`.
3. Place the manuscript, data, code, supplementary materials, preregistration, environment files, and authorized supporting sources in that directory.
4. Open **Explore Workflows** and select **micro1/ClaimBounty**.
5. Import the workflow. The export installs the workflows, schemas, templates, and runner needed by ClaimBounty.
6. Bind the `case-bundle` input to `claimbounty/input/case-bundle/`.
7. Review the inputs and run the parent workflow.

The `micro1/ClaimBounty` listing and import must be verified against the release build before publication. If the listing is unavailable, import the repository package under `workflow/claimbounty-scientific-audit/` by following [REPRODUCE.md](REPRODUCE.md). Study-specific Python, R, system-library, and data requirements are separate from the hosted application prerequisites above.
