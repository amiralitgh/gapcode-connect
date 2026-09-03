# Contributing

GapCode Connect is a focused fork of cc-connect. Changes should preserve
existing platform behavior while keeping Telegram and Bale safe and easy to
configure.

## Before submitting a change

1. Search existing issues and pull requests.
2. Keep credentials, local configuration, and runtime data out of commits.
3. Add or update tests for behavior changes.
4. Update the English documentation when configuration or user-visible behavior
   changes.

Run:

```bash
gofmt -w .
go test ./...
go test ./... -race
go build ./...
scripts/check-public-repo.sh
```

## Pull requests

Include a concise summary, affected integrations, test commands and results,
and security considerations for changes involving tokens, commands, files, or
network access.

Do not include bot tokens, API keys, personal chat IDs, local paths, or private
logs.

## Reporting security issues

Do not disclose credentials or exploitable details in a public issue. Follow
the process in [SECURITY.md](SECURITY.md).
