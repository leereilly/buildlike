# Security Policy

`gh-commit-crawl` is a single-binary, locally-run `gh` CLI extension. It
makes exactly one outbound HTTP request per `--user` invocation, to the
unauthenticated `https://github.com/<handle>.contribs` endpoint. It does
**not** read or write tokens, and it has no network listeners.

That said, attack surface is attack surface — please report anything that
looks like a vulnerability.

## Reporting a vulnerability

Please **do not open a public GitHub issue** for security reports.

1. Use [GitHub's private vulnerability reporting](https://github.com/leereilly/gh-commit-crawl/security/advisories/new)
   on this repository, **or**
2. Email **lee@leereilly.net** with the details.

We aim to:

- Acknowledge your report within **3 business days**.
- Provide a remediation plan or initial assessment within **10 business
  days**.
- Credit you in the release notes (if you wish) once a fix ships.

## Scope

In scope:

- The Go source under `cmd/` and `internal/`.
- The GitHub Actions workflows in `.github/workflows/`.
- The released extension binaries.

Out of scope:

- Vulnerabilities in the `gh` CLI itself — please report those upstream at
  [`cli/cli`](https://github.com/cli/cli/security).
- Vulnerabilities in GitHub.com's `.contribs` endpoint — please report those
  via [GitHub's bug bounty program](https://bounty.github.com/).
- Findings that require an attacker to already have write access to your
  shell (e.g., "if you pre-set `$PATH` to a malicious `gh`, things break").

## Supported versions

Only the latest released tag is supported. Please upgrade with
`gh extension upgrade commit-crawl` before reporting.
