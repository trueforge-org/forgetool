# Forgetool Copilot Instructions

- This repository is a Go CLI application (`forgetool`) with commands under `/cmd` and implementation packages under `/pkg`.
- Prefer minimal, surgical changes; do not refactor unrelated code.
- Keep changes idiomatic Go and run `gofmt` on touched Go files.
- Validate changes with targeted tests first, then `go test ./...` when practical.
- Do not add new dependencies unless absolutely required.
- Always use semantic commit names for commits and semantic titles for PRs.
- Keep generated or temporary files out of commits.
