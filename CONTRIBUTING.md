# Contributing Guide

Thank you for contributing to Google Search Console MCP. This guide outlines quality expectations and the contribution process.

## Quick Start

**Requirements:** Go 1.25+

```bash
make check
```

Run container integration checks with `make docker-test`. Tests use fixtures and never require Google credentials.

## Commits — Iterative and Atomic

Follow [ACDD](https://opatsay.com/ru/atomarnye-kommity-acdd):

- Keep changes **small and atomic**: one commit = one complete, verified outcome; include all files required for that outcome
- Every published commit must build and pass the checks present at that point
- Keep a bug fix and its regression test together; separate passing characterization tests are welcome
- Update mutually dependent tooling/configuration in the same commit
- Prefer one file per commit when that file delivers a self-contained, verifiable change
- Keep multiple files together only when they are required for the same complete step; explain that dependency in the commit body
- Split independent changes even when they touch the same file
- Write the intended step before editing, verify it, then commit before starting the next step
- Review every commit in order before merging
- Prefer small, focused pull requests

## Mandatory Comments and Tests

- Public types and non-trivial logic must include comments that explain the **why**
- New features **must include unit tests**; bug fixes should include a regression test
- Do not regress quality: run `make check` before pushing

## Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>[optional scope]: <imperative short summary>

[optional body]
```

| Type | Description |
|------|-------------|
| `feat` | New functionality |
| `fix` | Bug fixes |
| `refactor` | Refactoring without changing behavior |
| `perf` | Performance improvements |
| `test` | Add or fix tests |
| `chore` | Technical changes that do not change behavior |
| `docs` | Documentation |
| `style` | Formatting |
| `build` | Build system and dependencies |
| `ci` | CI/CD configuration |

### Examples

```text
feat(gsc): add Search Console domain types
test(gsc): cover analytics date validation
docs: describe remote MCP agent configuration
ci: run unit tests on push to main
```

## Branch Naming

- `feat/` — new features
- `fix/` — bug fixes
- `docs/` — documentation only
- `ci/` — CI configuration

## Security

- Never commit secrets, keys, or `.env` files
- Report security issues privately to: smmartbiz@gmail.com

## Documentation

- Update **README.md** for user-facing changes
- Keep godoc on exported types current

## Getting Help

- **Bugs:** open an issue with reproduction steps
- **Features:** open an issue before implementing large changes
- **Security:** email smmartbiz@gmail.com privately

## Code of Conduct

- Be respectful and constructive
- Welcome newcomers
- Focus on what is best for the community

Thank you for your contribution.
