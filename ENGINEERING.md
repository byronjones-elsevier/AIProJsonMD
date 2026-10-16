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
- **Standard library `testing`, not testify.** Matches the project copy of
  `agents/go.md` and the original tests. The global standard prefers testify;
  see Remaining Risks.
- **No `context.Context` parameters.** The tool does short, local file I/O
  with no cancellable work, so there is nothing to propagate.

## Verified

Checked on 2026-10-16 with Go 1.27.1 on macOS:

- `gofmt -l .` reports nothing.
- `go vet ./...` is clean.
- `golangci-lint run ./...` reports 0 issues (default linters).
- `go test -race -cover ./...` passes with 89% statement coverage.
- `make help`, `make all`, `make test/cover`, `make clean`, and the `make run`
  guard without `IN` behave as documented.
- `run` end to end on a synthetic export: folders, `Unfiled`, duplicate names,
  a missing blob, an image attachment, names with spaces and parentheses, and
  the overwrite and no-overwrite paths.

Not exercised:

- A run against a real, large export after the refactor. The pre-refactor
  output was not diffed against the new output.
- `govulncheck` (not installed on the development machine) and `make install`.
- Windows. Filename sanitizing covers its reserved characters but not reserved
  names such as `CON`.

## Remaining Risks and Possible Work

- `-assets` is joined into the output path without validation, so a value such
  as `../x` writes outside the output folder. The accepted validation rule has
  not been decided.
- Output files use mode `0644` and directories `0755`. Chat history may be
  sensitive; `0600` and `0700` may be preferable.
- Very long conversation names can exceed filesystem name limits and fail the
  write.
- The global Go standard prefers `testify`; the project currently uses the
  standard library. Decide which applies and align `agents/go.md`.
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
