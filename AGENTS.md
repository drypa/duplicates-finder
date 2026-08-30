# AGENTS.md

## What this is

CLI tool that finds duplicate files between a source and target directory. Compares by filename + file size + murmur3-128 hash (not content-equal across different names).

## Build & run

```bash
go build -o duplicates-finder .
./duplicates-finder --source-dir /path/src --target-dir /path/tgt --action print
```

No test suite exists. No lint/format tooling is configured.

## CLI flags

| Flag | Short | Required | Default | Notes |
|------|-------|----------|---------|-------|
| `--source-dir` | `-s` | yes | — | Directory to index as reference |
| `--target-dir` | `-t` | yes | — | Directory scanned for duplicates |
| `--action` | `-a` | no | `Nothing` | One of: `Nothing`, `Print`, `DeleteSource`, `DeleteTarget` |
| `--parallelism` | `-p` | no | `5` | Concurrency for file scanning/hashing |

`Delete*` actions prompt for confirmation (`y/n`) before deleting.

## Architecture

- `main.go` — entrypoint, delegates to `cmd`
- `cmd/cmd_find.go` — cobra command, file traversal, duplicate detection logic
- `files/file.go` — file metadata + murmur3-128 hashing (`spaolacci/murmur3`)
- `actions/action.go` — action enum (`Nothing`/`Print`/`DeleteSource`/`DeleteTarget`)

## Gotchas

- Module path is `duplicates-github.com/drypa/duplicates-finder` (unusual: has `github.com` in the middle, not just `github.com/drypa/...`). Import paths must match this exactly.
- Duplicate matching is by **filename** (basename), not by full path. Two files with the same name in different subdirectories will be compared.
- `sourceFiles` is a package-level `map[string]*files.File` — not reset between runs. Only matters if the code is used as a library (currently it's CLI-only).
- Parallelism validation: `parallelism <= 1` returns an error but the message says "greater or equal to 1" — the check is `<= 1`, so value `1` is rejected. Use `2+`.
