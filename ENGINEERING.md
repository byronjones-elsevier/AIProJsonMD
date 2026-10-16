# Engineering Notes

Handover notes for the engineer or agent continuing this work. Read them before
making changes, and update them in the same change set when architecture,
decisions, verification status, or known risks change.

## What This Is

`AIProJsonMD` is a single-binary Go CLI that converts an exported AI Pro chat
JSON file into:

- one Markdown file per conversation
- one output subfolder per export folder name, plus `Unfiled` for chats without
  a mapped folder
- one flat `assets/` directory inside each output folder holding extracted
  attachments, linked from the Markdown

Usage and flags are documented in [README.md](README.md).

## Tech Stack

- Go 1.26+, standard library only (no third-party dependencies)
- Module path `github.com/byronjones-elsevier/AIProJsonMD`
- GNU Make for tasks, `golangci-lint` for linting, `govulncheck` for
  vulnerability scans (the last two are optional tools, not dependencies)

## File Map

| File | Purpose |
| --- | --- |
| `main.go` | Flag parsing and `run(args, stdout) error`; `main` logs the error and exits 1 |
| `model.go` | Types for the export JSON: `Export`, `Conversation`, `Folder`, `Message`, `FileRef`, `FileBlob` |
| `convert.go` | Conversion: folder paths, Markdown file naming, message and attachment rendering |
| `files.go` | File helpers: filename sanitizing, no-clobber writes, base64 decoding, attachment payload and path resolution |
| `main_test.go` | Original tests for `writeFile`, `attachmentPayload`, `decodeBase64Payload`, `conversationMarkdownPath` |
| `files_test.go` | Table-driven tests for the `files.go` helpers |
| `convert_test.go` | Tests for `convert.go` helpers and end-to-end `run` tests on a synthetic export |
| `Makefile` | Build, test, lint, and help targets (see `make help`) |
| `Agents.md`, `agents/` | Agent guidance and the topic files it routes to |
| `.markdownlint.json` | Markdown lint configuration |
| `in_json/`, `out_markdown/` | Local input and generated output; both are gitignored |

## Architecture

`run` is the only orchestrator:

1. Parse flags (`-in`, `-out`, `-assets`, `-overwrite`) with a `flag.FlagSet`
   so `run` is testable.
2. Read and unmarshal the input into `Export`. An empty `history` is an error.
3. `newConverter` indexes top-level `files` by `fileKey` and builds the folder
   ID to output path map (sanitized names, numeric suffix for duplicates).
4. For each conversation, `convertConversation`:
   1. picks the folder from `folderId`, falling back to `Unfiled`
   2. picks a unique Markdown path from the conversation name
   3. renders the title, ID, and messages
   4. merges conversation-level and message-level attachment refs,
      de-duplicates them, and extracts each blob into `<folder>/assets/`
   5. writes the Markdown file
5. Print how many Markdown files were written.

Error handling: fatal errors are wrapped with `%w` and returned to `main`,
which logs them once with `slog` and exits 1. Per-attachment failures do not
stop the run; they are written into the Markdown as a note and logged with
`slog.Warn`.

## Schema Quirks Observed

- `fileSize` is a string in some places and a number in others, so it is typed
  `any`. It is not used.
- `folderId` can be `null`; those conversations go to `Unfiled`.
- Some conversations have empty `files` even when message-level attachments
  exist, so both sources are merged.
- Attachment payloads are in `files[].content` or `files[].imageBase64`. The
  base64 value may be a data URI, may contain line breaks, and may be unpadded.
- `fileType` helps infer an extension for images whose name has none.

## Key Decisions

- **Markdown names come from the conversation name.** Chosen in PRs #1 and #2.
  Rejected: hash plus index names, which are stable but unreadable. The hash
  form remains as the fallback for unnamed conversations.
- **Flat `assets/` per folder with collision handling.** Identical content is
  reused; different content with the same name gets `_2`, `_3`, ... Rejected:
  a subdirectory per conversation, which makes links longer and the tree deeper.
- **No overwrite by default.** Files are created with `O_EXCL`, which avoids a
  check-then-write race. `-overwrite` replaces them.
- **Fail fast on fatal errors, degrade on attachments.** Rejected: collecting
  every error into a summary, which adds complexity for a one-shot tool.
- **Standard library `testing`, not testify.** Decided deliberately: the
  project has no third-party dependencies, so there is no `go.sum` and almost
  nothing for `govulncheck` to scan beyond the standard library. The suite is
  table-driven and readable without assertion helpers. This is an intentional
  project-level override of the global `testify` preference. Rejected: adding
  `testify`, which would add the first dependency and rewrite about 60
  assertions for little gain.
- **`-assets` must be a single directory name.** Empty, `.`, `..`, absolute
  paths, path separators, and control characters are rejected with
  `ErrInvalidAssetsDir`. Rejected: allowing nested relative paths, which needs
  containment checks and complicates link generation with no known use.
- **Private output.** Files are created with mode `0600` and directories with
  `0700`, because chat history can be sensitive. This applies only to what the
  tool creates; existing files keep their mode (an existing file replaced with
  `-overwrite` keeps its current mode).
- **Names are capped at 200 bytes.** `sanitizeFilename` truncates on a UTF-8
  boundary and keeps a short file extension, leaving room for `.md` and `_N`
  suffixes under the 255-byte filesystem limit.
- **Pinned `govulncheck`.** `make vuln` runs
  `go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)` (currently
  `v1.9.0`) so nothing is installed globally and the version is reproducible.
- **No `context.Context` parameters.** The tool does short, local file I/O
  with no cancellable work, so there is nothing to propagate.

## Verified

Checked on 2026-10-16 with Go 1.27.2 and `golangci-lint` 2.15.0 on macOS:

- `gofmt -l .` reports nothing.
- `go vet ./...` is clean.
- `golangci-lint run ./...` reports 0 issues (default linters).
- `go test -race -cover ./...` passes with 88.8% statement coverage (89.7% on
  Go 1.27.1; the difference is the toolchain's instrumentation, not the tests).
- `make help`, `make all`, `make test/cover`, `make clean`, and the `make run`
  guard without `IN` behave as documented.
- `run` end to end on a synthetic export: folders, `Unfiled`, duplicate names,
  a missing blob, an image attachment, names with spaces and parentheses, the
  overwrite and no-overwrite paths, rejected `-assets` values, and the
  `0600`/`0700` output modes.
- `make vuln` (`govulncheck v1.9.0`) reports no vulnerabilities on Go 1.27.2.
  On Go 1.27.1 it reported 4 standard-library findings (GO-2026-6603, -6611,
  -6613, -6617) in `net/http` server and HTTP/2 code. The tool only calls
  `http.DetectContentType` and never starts an HTTP server or client, so those
  traces went through `init` and `io.Writer` dispatch rather than real HTTP
  use. Upgrading the toolchain cleared them. There are no third-party
  dependencies.
- Output of the pre-refactor build (from `main`) and the refactored build,
  both run on a real export of 40 conversations and diffed. The file trees
  matched (47 files each). One Markdown file differed: three attachment links
  with spaces in the name are now URL-escaped, as intended. All other Markdown
  and attachment files were byte-identical.
- Output of `main` before the hardening (`b99089e`) and the hardened build,
  both run on the same real 40-conversation export and diffed. All 47 files
  were byte-identical. The longest file name was 42 bytes in both, so the
  200-byte cap did not trigger. The only difference was permissions: every file
  went from `0644` to `0600`, and every directory from `0754` to `0700`.

Not exercised:

- `make install`.
- Windows. Filename sanitizing covers its reserved characters and reserved
  device names (`CON`, `NUL`, `COM1`, ... get a leading `_`) in unit tests, but
  the tool has not been run on Windows, and the permission test is skipped
  there.
- The 200-byte name cap on real data, since no name in the sample export is
  near the limit. It is covered by unit tests only.

## Remaining Risks and Possible Work

- `go.mod` sets `go 1.26` with `toolchain go1.27.2`, so `go` fetches a patched
  toolchain when the local one is older. Raise the `toolchain` line when a
  newer Go patch release fixes a standard-library vulnerability.
- `golangci-lint` must be built with a Go release at least as new as the
  project's toolchain. A 2.13.2 build made with Go 1.27.0 failed typechecking
  on Go 1.27.2 until it was upgraded.
- `make vuln` needs network access to the Go module proxy and vulnerability
  database, so it is deliberately not part of `make all`.
- There is no CI workflow and no pinned `golangci-lint` configuration.

## Build, Run, and Validate

```bash
make build                 # build ./AIProJsonMD
make run IN=export.json    # convert an export
make all                   # fmt/check, vet, lint, race tests, build
make test/cover            # tests with a coverage total
make vuln                  # govulncheck
npx --yes markdownlint-cli2 '**/*.md' '#.git/**' '#out_markdown/**'
```
