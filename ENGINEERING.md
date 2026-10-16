# Engineering Notes

Handover notes for the engineer or agent continuing this work. Read them before
making changes, and update them in the same change set when architecture,
decisions, verification status, or known risks change.

## What This Is

`AIProJsonMD` is a single-binary Go CLI that converts an exported AI Pro chat
JSON file into:

- one Markdown file per conversation (`-format md`, the default), one
  self-contained HTML file per conversation (`-format html`), or both
  (`-format both`)
- with HTML output, an `index.html` at the top of the output directory linking
  every folder and conversation, and an `index.html` inside each folder listing
  that folder's conversations; every page below the main index has a
  breadcrumb whose links go up one level and back to the main index
- one output subfolder per export folder name, plus `Unfiled` for chats without
  a mapped folder
- one flat `assets/` directory inside each output folder holding extracted
  attachments, linked from the Markdown and HTML

Usage and flags are documented in [README.md](README.md).

## Tech Stack

- Go 1.26+
- One runtime dependency, [`github.com/yuin/goldmark`](https://github.com/yuin/goldmark)
  (v1.8.6), used only to render message Markdown to HTML. Everything else is
  the standard library.
- Module path `github.com/byronjones-elsevier/AIProJsonMD`
- GNU Make for tasks, `golangci-lint` for linting, `govulncheck` for
  vulnerability scans (the last two are optional tools, not dependencies)

## File Map

| File | Purpose |
| --- | --- |
| `main.go` | Flag parsing and `run(args, stdout) error`; `main` logs the error and exits 1 |
| `model.go` | Types for the export JSON: `Export`, `Conversation`, `Folder`, `Message`, `FileRef`, `FileBlob` |
| `format.go` | The `-format` value: `outputFormat`, `parseFormat`, `ErrInvalidFormat` |
| `convert.go` | Converter setup and helpers: folder paths, file naming, link escaping |
| `conversation.go` | Per-conversation work: attachment extraction, Markdown rendering, writing Markdown, HTML, and the index |
| `html.go` | HTML rendering: page and index templates, the goldmark renderer, raw-HTML escaping |
| `files.go` | File helpers: filename sanitizing, no-clobber writes, base64 decoding, attachment payload and path resolution |
| `main_test.go` | Original tests for `writeFile`, `attachmentPayload`, `decodeBase64Payload`, `conversationMarkdownPath` |
| `files_test.go` | Table-driven tests for the `files.go` helpers |
| `convert_test.go` | Tests for `convert.go` helpers and end-to-end Markdown `run` tests on a synthetic export |
| `html_test.go` | HTML tests: Markdown rendering, raw-HTML and URL safety, page and index output, `-format` runs |
| `nav_test.go` | Navigation tests: breadcrumb and up/home links, folder pages, every local link in a generated site resolving, reserved `index` names |
| `go.mod`, `go.sum` | Module definition with the goldmark dependency and its checksums |
| `Makefile` | Build, test, lint, and help targets (see `make help`) |
| `Agents.md`, `agents/` | Agent guidance and the topic files it routes to |
| `.markdownlint.json` | Markdown lint configuration |
| `.golangci.yml` | `golangci-lint` v2 configuration: standard linters plus `errorlint`, `gosec`, `misspell`, `revive`, `unconvert` |
| `.github/workflows/ci.yml` | CI: tests on Linux and macOS, lint, `govulncheck`, Markdown lint |
| `.github/dependabot.yml` | Weekly updates for pinned action SHAs and the `go`/`toolchain` directives |
| `in_json/`, `out_markdown/` | Local input and generated output; both are gitignored |

## Architecture

`run` is the only orchestrator:

1. Parse flags (`-in`, `-out`, `-assets`, `-format`, `-overwrite`) with a
   `flag.FlagSet` so `run` is testable, and validate `-assets` and `-format`.
2. Read and unmarshal the input into `Export`. An empty `history` is an error.
3. `newConverter` indexes top-level `files` by `fileKey`, builds the folder
   ID to output path map (sanitized names, numeric suffix for duplicates), and
   creates the HTML renderer when the format includes HTML.
4. For each conversation, `convertConversation`:
   1. picks the folder from `folderId`, falling back to `Unfiled`
   2. picks a unique base name from the conversation name; the `.md` and
      `.html` files share it
   3. merges conversation-level and message-level attachment refs,
      de-duplicates them, and extracts each blob into `<folder>/assets/`
      once, producing a list of `attachment` values
   4. renders and writes the Markdown file (`renderMarkdown`) and/or the HTML
      file (`htmlRenderer.renderConversation`) from the same data, and records
      an index entry for the HTML file
5. `writeIndexes` renders the main `index.html` from the collected entries and
   an `index.html` for each folder (HTML formats only). Each conversation and
   folder page gets its navigation from `conversationNav` or `folderNav`, with
   the path back to the main index computed by `homeHref`.
6. Print how many files of each format were written.

Error handling: fatal errors are wrapped with `%w` and returned to `main`,
which logs them once with `slog` and exits 1. Per-attachment failures do not
stop the run; they are written into the output as a note and logged with
`slog.Warn`.

### HTML rendering and safety

Chat content is untrusted, so the HTML path is built to stay safe even if one
layer fails:

- Page structure uses `html/template`, which escapes every interpolated value
  (titles, IDs, roles, attachment names, failure notes).
- Message bodies go through goldmark with the Table, Strikethrough, and
  TaskList extensions. goldmark is used in its default safe mode, so links and
  images with `javascript:`, `vbscript:`, `file:`, and non-image `data:` URLs
  lose their `href` or `src`.
- Raw HTML in a message is shown as escaped text, not passed through and not
  dropped. goldmark's safe mode alone would replace it with
  `<!-- raw HTML omitted -->` and lose the text, so `escapeRawHTML` overrides
  the two node renderers (`RawHTML`, `HTMLBlock`).
- The role shown on a message never reaches a CSS class directly; `roleClass`
  maps it to a fixed set.
- Each page carries a `Content-Security-Policy` meta tag
  (`default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:;
  base-uri 'none'`) and contains no JavaScript, so scripts and external
  resources are blocked even if an escaping bug appeared.
- Image attachments are shown inline from the local `assets/` directory.
  External images referenced in Markdown are blocked by the policy.

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
- **Standard library `testing`, not testify.** Decided deliberately: the suite
  is table-driven and readable without assertion helpers, and the project
  keeps its dependency set to what the product needs. This is an intentional
  project-level override of the global `testify` preference. Rejected: adding
  `testify`, which would add a test-only dependency and rewrite about 60
  assertions for little gain.
- **HTML export uses goldmark.** Chosen over a hand-written renderer, which
  would stay dependency-free but would need to reproduce fenced code, nested
  lists, and tables correctly, and over escaped plain text, which is the safest
  but hard to read. Real chat history is heavy on all three: in the sample
  export, about 30% of messages had fenced code and nearly half had lists. This
  ended the earlier "no third-party dependencies" position, so `go.sum`,
  Dependabot's `gomod` updates, and `govulncheck` now cover a real dependency.
- **`-format md|html|both`, default `md`.** Existing invocations behave exactly
  as before. Rejected: a boolean `-html` flag, which cannot produce HTML only.
- **Three-level HTML hierarchy with navigation.** The main `index.html` lists
  every conversation grouped by folder (alphabetical, `Unfiled` last), each
  folder heading links to a folder page, and every conversation and folder page
  shows a breadcrumb (`Main index > folder > conversation`). The breadcrumb is
  the only navigation: the crumb before the current page is the way up one level
  and the first crumb is the way back to the main index. An earlier version also
  rendered a second line (`Up to <folder> · Main index`); it only repeated the
  breadcrumb's links, so it was removed after review. The folder page exists so
  "up one level" from a conversation has a real target. The current page is the
  last crumb, unlinked and marked `aria-current="page"`. All links are relative,
  so the tree works from disk or after being moved. These pages are only written for `html` and `both`. Like
  other output files they are kept, not rewritten, when they already exist
  unless `-overwrite` is set, so re-running with new conversations needs
  `-overwrite` to refresh them. Rejected: a breadcrumb that only links to the
  main index, which gives no way up one level.
- **`index` is a reserved conversation name.** Each folder's page is
  `index.html`, so a conversation titled "index" is written as `index_2`. The
  reservation applies to every format, so Markdown and HTML names always match.
  The cost is that a Markdown-only run also renames a conversation titled
  "index".
- **Folder directory names are unique ignoring case.** `buildFolderPaths` now
  compares names case-insensitively and reserves `index.html` (the main index
  file), so `Docs` and `docs` become `Docs` and `docs_2` rather than sharing a
  directory on case-insensitive filesystems. It also fixes a latent collision
  when a folder was literally named like an already-suffixed one (`Docs_2`).
- **One extraction, two renderers.** Attachments are extracted once into a
  neutral `attachment` list that both renderers read, so the Markdown output
  stays byte-identical to the pre-HTML version and the two formats cannot
  disagree about file names.
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
- `golangci-lint run ./...` reports 0 issues (standard linters plus
  `errorlint`, `gosec`, `misspell`, `revive`, `unconvert`).
- `go test -race -cover ./...` passes with 89.6% statement coverage.
- `make help`, `make all`, `make test/cover`, `make clean`, and the `make run`
  guard without `IN` behave as documented.
- `run` end to end on a synthetic export: folders, `Unfiled`, duplicate names,
  a missing blob, an image attachment, names with spaces and parentheses, the
  overwrite and no-overwrite paths, rejected `-assets` values, and the
  `0600`/`0700` output modes.
- `.github/workflows/ci.yml` passes `actionlint`, every action is pinned to a
  full commit SHA, and each CI step (`make fmt/check vet test/race build`, the
  lint config, and the pinned markdownlint command) was run locally. The
  workflow itself has not yet run on GitHub; the first pull request will be its
  first real run.
- `make vuln` (`govulncheck v1.9.0`) reports no vulnerabilities on Go 1.27.2.
  On Go 1.27.1 it reported 4 standard-library findings (GO-2026-6603, -6611,
  -6613, -6617) in `net/http` server and HTTP/2 code. The tool only calls
  `http.DetectContentType` and never starts an HTTP server or client, so those
  traces went through `init` and `io.Writer` dispatch rather than real HTTP
  use. Upgrading the toolchain cleared them. The scan was repeated after adding
  goldmark and still reports no vulnerabilities.
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
- `main` before HTML export (`796b2bc`) versus the HTML build, run on the same
  real export (`-format both` for the new build). Markdown content and
  attachments were byte-identical in all 47 files. Five Markdown files were
  renamed: conversation names that ended in a space (truncated chat titles
  such as "Please review this confluence") lost the trailing space, which
  Windows does not allow. The content of all five was identical. This comes
  from the Windows-name change, not from HTML; re-running into an existing
  output directory without `-overwrite` leaves the old and the new name side by
  side.
- Generated HTML on the real export: 40 conversation pages plus `index.html`.
  An audit found no `script`, `iframe`, `object`, `embed`, `form`, `base`, or
  `link` elements and no event-handler attributes; every page has the
  Content-Security-Policy meta tag; all 98 link and image targets that point at
  local files exist. The only URL schemes present were `https` (11), `mailto`
  (36, from email autolinks), and relative paths. 4 image attachments rendered
  inline. 5 tables, 214 code blocks, and 1,551 list items were rendered.

- Navigation on the real export (`-format both`): the Markdown and attachments
  were byte-identical to `main` (`ab33ef0`). The HTML had 1 main index, 9 folder
  pages, and 40 conversation pages. Every folder and conversation page had a
  breadcrumb with exactly one current-page crumb (2 crumbs on folder pages, 3 on
  conversation pages), a first crumb linking to the main index, and, on
  conversation pages, a parent crumb linking to the folder page. No page linked
  to the main index anywhere except the breadcrumb, and none kept the removed
  second navigation line. The main index had no breadcrumb. No local link was
  unresolved.

Not exercised:

- Viewing the pages in a browser. Structure, safety, navigation, and link
  integrity were checked programmatically and by unit tests, but the visual
  styling (breadcrumb layout, dark mode, wide tables, long code lines) has not
  been looked at.

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
- CI has no Windows job. Add `windows-latest` to the test matrix once the tests
  have been confirmed there.
- `govulncheck` runs in CI on every push and pull request, so a newly published
  advisory can fail an unrelated PR. Raise the `go.mod` `toolchain` line or
  `GOVULNCHECK_VERSION` to clear it.

## Build, Run, and Validate

```bash
make build                 # build ./AIProJsonMD
make run IN=export.json    # convert an export
make all                   # fmt/check, vet, lint, race tests, build
make test/cover            # tests with a coverage total
make vuln                  # govulncheck
npx --yes markdownlint-cli2 '**/*.md' '#.git/**' '#out_markdown/**'
```
