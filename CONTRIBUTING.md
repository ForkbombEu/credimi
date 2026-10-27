<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV
SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# Contributing to Credimi

Thanks for your interest. The full rules for maintainers and agents live in
[`AGENTS.md`](./AGENTS.md); this file covers the essentials for external
contributors. See the [documentation](https://docs.credimi.io/) for product
details.

## Before you start

- Search [issues](../../issues) and [discussions](../../discussions) to avoid
  duplicate work.
- For a bug, open an issue with steps to reproduce and the version you ran.
- For a larger change, open an issue or discussion first so we can agree on the
  approach before you invest time.

## Development setup

Follow the [developer setup guide](https://docs.credimi.io/). In short:

```sh
make dev            # infrastructure + API + UI
make dev.noworkers  # same stack without Temporal workers (faster boot)
```

## Checks before submitting

Go changes:

```sh
make fmt
make lint
make test
```

Webapp changes (`webapp/`):

```sh
cd webapp && bun run lint && bun run check
```

## Submitting a pull request

1. Create a branch named `<type>/<scope>-<short-description>` (for example
   `fix/queue-cancel-cleanup`).
2. Keep the change small and focused.
3. Use Conventional Commit messages; the subject is imperative and lowercase.
4. Fill in the pull request template.

CI runs formatting, linting, vet, and tests. Make sure they pass before
requesting review.

## Licence

By contributing, you agree that your contributions are licensed under the same
licence as the project. Credimi is licensed under the
[GNU Affero General Public License v3.0 or later](./LICENSE).
