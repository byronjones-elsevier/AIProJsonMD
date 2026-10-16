# Changelog

A record of every unit of work on `AIProJsonMD`: each branch and pull request,
what it did, when, and by whom. Newest first. It follows the standard in
[agents/coding.md](agents/coding.md). Specification: [ENGINEERING.md](ENGINEERING.md).
Outstanding work: [HANDOFF.md](HANDOFF.md).

Conventions:

- Timestamps are ISO 8601 in UTC (`Z`). Commit times come from git author
  dates; PR times come from GitHub.
- Start an entry when a branch is created, keep it current as the work
  proceeds, and finalize it before the pull request is opened.
- Correct an entry with a dated note rather than rewriting it.
- This file was created on 2026-10-16 and backfilled from the git history and
  GitHub pull request metadata.

## People and tools

- **Byron Jones**: GitHub `byronjones-elsevier`. Git identities:
  `ByronJones-Elsevier <b.jones1@elsevier.com>` for commits made locally, and
  `Byron Scott Jones @ Elsevier` for commits and merges made through the GitHub
  web interface. Owner of the project. Directed the work, reviewed it, pushed
  every branch from his own terminal, and merged every pull request.
- **Claude Sonnet 5.5, via Claude Code** (Anthropic). Worked in Byron's
  checkout on 2026-10-16, from PR #3 onward. Authored the commits under Byron's
  local git identity (commit trailer "Developed with the assistance of
  Claude.ai"), opened pull requests through Byron's authenticated `gh`, and
  did not push branches. Referred to below as "Claude".
- **GitHub Copilot Autofix**: credited by a commit trailer on three commits of
  PR #1.
- Where a commit carries no AI trailer and the history does not say, AI
  assistance is "not recorded".

## In flight

### docs/changelog-pr20-merged: record PR #20 as merged (no PR yet)

- Branch: `docs/changelog-pr20-merged`, base `main` (`0a37544`); PR not yet
  opened; commits not yet made
- Status: open
- Started: 2026-10-16T17:47:34Z (branch created from `main`; first commit follows)
- Worked on by: Byron Jones (reported the merge; will push and merge); Claude
  Sonnet 5.5 via Claude Code (author)
- What it does: moves the PR #20 entry from "In flight" to History with its
  merge time, states the incident above, brings `HANDOFF.md` up to date with
  `main` at `0a37544`, and updates the "last reviewed" commit in
  `ENGINEERING.md`.
- Spec and docs: `CHANGELOG.md`, `HANDOFF.md`, `ENGINEERING.md` header only.
- Verification: documentation only; `make all` and markdownlint run before commit.

## History

### docs/spec-handoff-changelog: specification, handoff, changelog, and reserved Unfiled (PR #20)

- Branch: `docs/spec-handoff-changelog`, base `main` (`590c032`); PR #20; commits
  `2bbae52..b0a4831` (3 commits; merge commit `0a37544`)
- Status: merged
- Started: 2026-10-16T17:30:09Z (first commit; the branch was created from `main`
  after PR #19 merged at 17:03:46Z)
- Last commit: 2026-10-16T17:37:23Z (`b0a4831`)
- Merged: 2026-10-16T17:45:59Z
- Worked on by: Byron Jones (requested the change and the three-file model,
  chose to reserve `Unfiled`, pushed, merged); Claude Sonnet 5.5 via Claude Code
  (author, opened the PR)
- What it did:
  - Refactors the global standard `~/.claude/agents/coding.md` (outside this
    repository) so `ENGINEERING.md` is the primary specification, `HANDOFF.md`
    tracks outstanding work, and `CHANGELOG.md` tracks all work.
  - Rewrites `ENGINEERING.md` as a requirements-based specification with
    stable IDs, exact algorithms, a worked example generated from real output,
    and the literal stylesheet, templates, and script as appendices.
  - Adds `HANDOFF.md` and this `CHANGELOG.md`.
  - Adds tests that tie the specification to the code: `spec_test.go` fails if
    the appendices (stylesheet, templates, script), the base policy, the
    dependency versions, or the reserved-name list drift, if section 11 names a
    test that does not exist, or if the worked example in section 4.5 no longer
    matches real output; `run_test.go` adds a determinism test (REQ-OUT-4) and a
    `-h` test (REQ-CLI-5).
  - Syncs `agents/coding.md`; updates `AGENTS.md` and `README.md` (including a
    corrected description of `-overwrite`).
  - Reserves the folder name `Unfiled` (decided by Byron after the reviews
    exposed the gap): an export folder named `Unfiled` in any capitalization now
    gets a numeric suffix (`Unfiled_2`), like any other name conflict, so the
    `Unfiled` directory holds only conversations without a mapped folder. This
    is a behavior change for exports that have a folder with that name. Code in
    `convert.go`, tests in `convert_test.go` and `nav_test.go`, and
    `ENGINEERING.md` (REQ-NAME-6, section 5.2, decision D-14).
- Spec and docs: all of `ENGINEERING.md`; `AGENTS.md`; `README.md`; `agents/coding.md`.
- Verification: `make all` passes (0 lint issues) and markdownlint exits 0. The
  drift test and the worked-example test were each shown to fail when the
  specification was altered and to pass when restored. The specification was
  reviewed in two rounds by fresh AI reviewers who were given only
  `ENGINEERING.md` and told not to read the code. Both rounds returned
  "mostly". Round 1 found gaps and two real contradictions (`-overwrite` versus
  attachments, and `Unfiled` versus case-insensitive folder uniqueness). Round 2
  found smaller gaps and several inconsistencies introduced by the round 1
  fixes. Every claim was checked against the code before acting. Findings were
  fixed in the specification, turned into tests, or recorded as open decisions
  (HANDOFF.md, H-07). `make all` passes with 0 lint issues and 90.7%
  coverage; markdownlint exits 0.
- Notes: the global standard edit is in `~/.claude/agents` and is not part of
  this repository's history. The reviews found places where the specification
  disagreed with the code (the `-overwrite` description, dangerous URLs
  rendered with an empty `href` rather than none, top-level `null` input, the
  `Unfiled` rule). The specification was corrected to match the code, except for
  `Unfiled`, which the owner then decided to change in the code (below).
  The new tests also turned five requirements that were "verified by review"
  (no deletion, no network calls, output stays inside `-out`, relative links,
  warning log format) into automated checks. One of the new drift tests caught
  an omission made while editing the specification itself.
- CI: all five jobs passed on the PR (tests on Linux and macOS, lint,
  vulnerability scan, Markdown lint), plus CodeQL.
- Incident: the PR was merged before a small follow-up commit (`602ffb7`) that
  recorded the PR number and range in this entry was pushed, so for one PR the
  changelog on `main` said this work was still open. The correction is the
  entry below (a documentation-only follow-up PR). Lesson recorded in
  HANDOFF.md section 7.

### docs/rename-agents-md: rename Agents.md to AGENTS.md (PR #19)

- Branch: `docs/rename-agents-md`, base `main`; PR #19; commit `a2b0e28`
- Status: merged
- Started: 2026-10-16T16:59:04Z
- Last commit: 2026-10-16T16:59:04Z
- Merged: 2026-10-16T17:03:46Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author, opened the PR)
- What it did: renamed the root file to `AGENTS.md` with a two-step `git mv`,
  updated every reference, and replaced `agents/coding.md` with the corrected
  global copy (the two files are byte-identical).
- Spec and docs: `README.md`, `ENGINEERING.md` file map, and `agents/` files.
- Verification: markdownlint exit 0; `make all` passed; all 10 relative links
  resolved with exact case; dry-run merge against `main` was clean; CI passed;
  after merge, `main` had no `Agents.md` references.
- Notes: this re-opens the change from PR #18 after it failed to reach `main`.

### docs/rename-agents-md: first attempt, stacked on the restore branch (PR #18)

- Branch: `docs/rename-agents-md`, base `docs/restore-line-break-note`; PR #18; commit `a2b0e28`
- Status: superseded by PR #19
- Started: 2026-10-16T16:59:04Z
- Merged: 2026-10-16T17:00:35Z (into its stack base, not into `main`)
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author, opened the PR)
- What it did: the same change as PR #19.
- Notes: PR #17 (the stack base) merged to `main` first, then #18 merged into
  the still-existing base branch, which was then deleted, so the rename never
  reached `main`. Found by checking `main` after the merge. Lesson recorded in
  HANDOFF.md: merge the base PR, delete its branch, and confirm the next PR
  retargeted to `main` before merging it.

### docs/restore-line-break-note: restore README sentence and fix lint (PR #17)

- Branch: `docs/restore-line-break-note`, base `main`; PR #17; commit `61d4822`
- Status: merged
- Started: 2026-10-16T16:45:14Z
- Last commit: 2026-10-16T16:45:14Z
- Merged: 2026-10-16T17:00:22Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author, opened the PR)
- What it did: restored the README sentence about line breaks and removed a
  stray blank line, both damaged when the README conflict between PRs #15 and
  #16 was resolved in the GitHub web editor.
- Verification: markdownlint exit 0; `make all` passed; CI on `main` went from
  failing (Markdown lint job) to passing, all 5 jobs.
- Notes: CI on `main` at `fdb4da1` had failed only the Markdown lint job.

### feat/copy-buttons: copy buttons on code blocks (PR #16)

- Branch: `feat/copy-buttons`, base `main`; PR #16; commits `42dbe3d..351e146`
- Status: merged
- Started: 2026-10-16T16:34:47Z
- Last commit: 2026-10-16T16:43:43Z
- Merged: 2026-10-16T16:43:57Z
- Worked on by: Byron Jones (requested the feature, pushed, resolved the merge
  conflict in the GitHub web editor, merged); Claude Sonnet 5.5 via Claude Code (author, tested in Chrome)
- What it did:
  - Added a Copy button to every code block, added by a small script, with
    Clipboard API use and an `execCommand` fallback, visible and spoken feedback,
    keyboard access, and hiding when printing.
  - Kept the page policy tight: the script is on code-block pages only and is
    allowed by SHA-256 hash; no `unsafe-inline` for scripts.
- Spec and docs: copy-button design and policy in `ENGINEERING.md`; README "Copy buttons".
- Verification: `make all`; script passed `node --check`; tests check the served
  script equals the constant, the policy hash matches, only code pages carry it,
  and chat content cannot add a script. On the real export all 26 code-block
  pages carried the script with a matching hash. In Chrome, from disk, a click
  replaced a clipboard marker with the exact block text and the button showed
  "Copied!". Not exercised: other browsers, the fallback path, the failure state,
  keyboard and screen reader use, dark mode, scripts disabled.
- Notes: the conflict resolution in the web editor dropped a README sentence
  and left a lint error; fixed by PR #17. The commit `351e146` is that web merge.

### fix/html-line-breaks: keep single newlines as line breaks (PR #15)

- Branch: `fix/html-line-breaks`, base `main`; PR #15; commit `3448220`
- Status: merged
- Started: 2026-10-16T16:23:58Z
- Last commit: 2026-10-16T16:23:58Z
- Merged: 2026-10-16T16:39:58Z
- Worked on by: Byron Jones (reported via screenshots, pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: enabled hard wraps in the Markdown renderer so a single newline
  in a message shows as a line break, as typed.
- Verification: tests for the behavior and its boundaries; on the real export
  the affected message showed two lines, and Byron confirmed in Chrome after a
  rebuild. 52 of 177 messages contained lines plain Markdown would have joined.
- Notes: found by viewing a generated page in Chrome through the `uictl` tools.

### feat/html-breadcrumbs: breadcrumbs and folder pages (PR #14)

- Branch: `feat/html-breadcrumbs`, base `main`; PR #14; commits `af4ae94..bc933ce`
- Status: merged
- Started: 2026-10-16T15:53:17Z
- Last commit: 2026-10-16T16:12:12Z
- Merged: 2026-10-16T16:17:17Z
- Worked on by: Byron Jones (requested, reviewed a screenshot, pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did:
  - Added a breadcrumb (`Main index > Folder > Conversation`) to every page
    below the main index, and an `index.html` in each folder.
  - Reserved the name `index` per folder; made folder directory names unique
    ignoring case and reserved `index.html` at the top level (also fixing a
    collision with a folder named like an already-suffixed one).
  - After Byron's screenshot review, removed a second "Up to folder, Main index"
    line that repeated the breadcrumb.
- Verification: `make all`; navigation, escaping, and link-integrity tests. On
  the real export: 1 main index, 9 folder pages, 40 conversation pages, every
  breadcrumb correct, all local links resolved, Markdown identical to `main`.
- Notes: the "Main index" link target on folder pages and conversation pages is
  computed relative to each page.

### feat/html-export: HTML output with index (PR #13)

- Branch: `feat/html-export`, base `main`; PR #13; commits `0b4feee..643fc4c`
- Status: merged
- Started: 2026-10-16T15:39:11Z
- Last commit: 2026-10-16T15:40:51Z
- Merged: 2026-10-16T15:46:00Z
- Worked on by: Byron Jones (requested the feature and chose goldmark, `-format`, and an index; pushed; merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: added `-format md|html|both` (default `md`), standalone HTML pages
  rendered with goldmark, raw HTML shown as escaped text, dangerous URLs removed,
  a Content-Security-Policy blocking scripts and external resources, inline
  image attachments, and a main `index.html`. First third-party dependency
  (`github.com/yuin/goldmark v1.8.6`).
- Spec and docs: HTML sections, safety model, and decisions in `ENGINEERING.md`; README.
- Verification: `make all`; 90.2% coverage; `govulncheck` clean. On the real
  export: 40 conversation pages plus an index, no script-capable elements, no
  event handlers, every page had the policy, all local links resolved; Markdown
  identical to `main` except five renamed files (from PR #11, see there).
- Notes: CI first ran on GitHub on `main` after these merges. The run for the
  #12 merge was cancelled by the #13 merge (its lint, vulnerability, and
  Markdown jobs had passed); the run for the #13 merge passed all 5 jobs.

### ci/github-actions: CI, lint config, Dependabot (PR #12)

- Branch: `ci/github-actions`, base `main`; PR #12; commit `10cd390`
- Status: merged
- Started: 2026-10-16T15:27:13Z
- Last commit: 2026-10-16T15:27:13Z
- Merged: 2026-10-16T15:45:13Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: added the CI workflow (tests on Linux and macOS, lint,
  `govulncheck`, Markdown lint; actions pinned to commit SHAs, read-only
  permissions, concurrency cancel, timeouts), `.golangci.yml` with extra linters,
  and Dependabot for actions and Go modules. Tightened test fixture file modes.
- Verification: `actionlint` clean; each CI step run locally. It had not run on
  GitHub when the PR merged (see PR #13 for the first runs).
- Notes: Windows is not in the test matrix because the tests were not confirmed
  there. CI does not run on stacked PRs targeting a non-`main` base.

### fix/windows-reserved-names: Windows-safe names (PR #11)

- Branch: `fix/windows-reserved-names`, base `main`; PR #11; commit `e2c690a`
- Status: merged
- Started: 2026-10-16T15:27:02Z
- Last commit: 2026-10-16T15:27:02Z
- Merged: 2026-10-16T15:44:20Z
- Worked on by: Byron Jones (requested, pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: prefixes `CON`, `PRN`, `AUX`, `NUL`, `COM1-9`, `LPT1-9` with `_`,
  and trims trailing spaces and dots from names.
- Verification: table-driven tests; not run on Windows.
- Notes: behavior change found later on the real export: 5 of 40 Markdown files
  were renamed because chat titles ending in a space lost that space (content
  identical). Byron chose to keep the trimming.

### docs/fix-security-md-lint: fix SECURITY.md lint (PR #10)

- Branch: `docs/fix-security-md-lint`, base `main`; PR #10; commit `cfff9e9`
- Status: merged
- Started: 2026-10-16T15:24:50Z
- Last commit: 2026-10-16T15:24:50Z
- Merged: 2026-10-16T15:43:52Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: removed a double blank line in `SECURITY.md` (markdownlint MD012).
- Verification: markdownlint clean.

### feat/harden-output-and-pin-vuln: hardening and pinned govulncheck (PR #9)

- Branch: `feat/harden-output-and-pin-vuln`, base `main`; PR #9; commits `8859d02..4704ff2`
- Status: merged
- Started: 2026-10-16T15:11:48Z
- Last commit: 2026-10-16T15:18:29Z
- Merged: 2026-10-16T15:23:01Z
- Worked on by: Byron Jones (decided the four design choices, upgraded Go and the linter, pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: `-assets` must be a single directory name; names capped at 200
  bytes; output files `0600` and directories `0700`; `make vuln` runs a pinned
  `govulncheck`; `toolchain go1.27.2` pinned in `go.mod`; stdlib-testing
  decision recorded.
- Verification: `make all`. `govulncheck` first reported 4 standard-library
  findings on Go 1.27.1 (GO-2026-6603, -6611, -6613, -6617), cleared by Go
  1.27.2; `golangci-lint` 2.13.2 failed on Go 1.27.2 until upgraded to 2.15.0.
  On the real export, output was byte-identical to `main` except modes
  (`0644` to `0600`, directories `0754` to `0700`).
- Notes: the sandbox blocked the Go and npm cache directories; caches were
  redirected to a temporary directory.

### docs/refresh-docs-and-agents: README, ENGINEERING.md, agent files (PR #8)

- Branch: `docs/refresh-docs-and-agents`, base `main`; PR #8; commits `8db40fc..a11ad4c`
- Status: merged
- Started: 2026-10-16T13:36:21Z
- Last commit: 2026-10-16T13:38:57Z
- Merged: 2026-10-16T15:04:44Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: rewrote `ENGINEERING.md` to the handover-notes structure, updated
  the README, synced `agents/coding.md` (adding the Engineering Handover Notes
  section) and added `makefile.md` and `.markdownlint.json`.
- Verification: markdownlint clean; links resolved.
- Notes: this PR was part of the first stack (#3 to #8), created together.

### build/makefile-standards: Makefile aligned with the standard (PR #7)

- Branch: `build/makefile-standards`, base `main`; PR #7; commit `afa5c89`
- Status: merged
- Started: 2026-10-16T13:35:12Z
- Last commit: 2026-10-16T13:35:12Z
- Merged: 2026-10-16T15:04:02Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: default `help` goal with the standard awk help, `?=` variables,
  `deps`, `all`, `default`, `install`, `fmt/check`, `lint`, `test/race`,
  `test/cover`, `vuln`, and a single-line guard on `run`; ignored `coverage.out`.
- Verification: `make help`, `make all`, `make test/cover`, `make clean`, and the
  `run` guard were run.
- Notes: `make clean` removed the old, git-ignored `AIProJsonMD` binary; it was
  rebuilt.

### test/expand-coverage: table-driven and end-to-end tests (PR #6)

- Branch: `test/expand-coverage`, base `main`; PR #6; commit `4b2838b`
- Status: merged
- Started: 2026-10-16T13:34:13Z
- Last commit: 2026-10-16T13:34:13Z
- Merged: 2026-10-16T15:02:58Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: tests for the file helpers and conversion helpers, plus `run()`
  end to end on a synthetic export and its error paths.
- Verification: coverage 27% to 89%; `go test -race` passed.

### refactor/run-and-split-files: run() and file split (PR #5)

- Branch: `refactor/run-and-split-files`, base `main`; PR #5; commit `9e60eb5`
- Status: merged
- Started: 2026-10-16T13:33:06Z
- Last commit: 2026-10-16T13:33:06Z
- Merged: 2026-10-16T15:02:19Z
- Worked on by: Byron Jones (pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: split `main.go` into `model.go`, `files.go`, `convert.go`; `main`
  calls `run(args, stdout) error`; errors wrapped and logged once with `slog`;
  sentinel errors; URL-escaped attachment links; the summary counts files
  actually written; `O_EXCL` writes; godoc on exported types.
- Verification: `make all`. A comparison of the old and new output on the real
  export was first declined at the permission prompt, then run step by step:
  the file trees matched (47 files) and one Markdown file differed, in three
  attachment links that are now URL-escaped, as intended.

### chore/go-module-path: real module path and Go version (PR #4)

- Branch: `chore/go-module-path`, base `main`; PR #4; commit `684e623`
- Status: merged
- Started: 2026-10-16T13:31:50Z
- Last commit: 2026-10-16T13:31:50Z
- Merged: 2026-10-16T15:01:21Z
- Worked on by: Byron Jones (confirmed the module path, pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: module path `github.com/byronjones-elsevier/AIProJsonMD`;
  `go 1.22` (unsupported) to `go 1.26`.
- Notes: the git remote still pointed at the former repository name
  (`AIChatJsonMD`) when the branches were pushed. Claude corrected the URL to
  the renamed repository, and Byron switched the remote to SSH because git over
  HTTPS did not work through the proxy.

### fix/gofmt-and-lint: formatting and lint fixes (PR #3)

- Branch: `fix/gofmt-and-lint`, base `main`; PR #3; commit `5eb1bc0`
- Status: merged
- Started: 2026-10-16T13:30:01Z
- Last commit: 2026-10-16T13:30:01Z
- Merged: 2026-10-16T15:00:56Z
- Worked on by: Byron Jones (requested the review, pushed, merged); Claude Sonnet 5.5 via Claude Code (author)
- What it did: fixed one `gofmt` indentation error and six `QF1012`
  (`WriteString(fmt.Sprintf(...))`) findings. No behavior change.
- Verification: `gofmt`, `go vet`, `golangci-lint` (0 issues), `go test -race`.
- Notes: the first change of the review of the project against Byron's Go,
  documentation, and Makefile standards.

### SECURITY.md: security policy (direct commit to main)

- Commit `af315df`, directly on `main`
- Status: merged
- Committed: 2026-10-16T13:18:18Z
- Worked on by: Byron Jones (through the GitHub web interface); AI assistance not recorded
- What it did: created `SECURITY.md` (supported versions and how to report
  vulnerabilities).
- Notes: the file had a markdownlint error, fixed in PR #10.

### use-conversation-names-for-markdown-files: add .gitignore (PR #2)

- Branch: `use-conversation-names-for-markdown-files` (reused), base `main`; PR #2; commit `01ccb04`
- Status: merged
- Started: 2026-10-16T13:14:03Z
- Merged: 2026-10-16T13:15:10Z
- Worked on by: Byron Jones; AI assistance not recorded
- What it did: added `.gitignore` (macOS files, the binary, `in_json/`,
  `out_markdown/`, `out_assets/`).

### use-conversation-names-for-markdown-files: conversation names for files (PR #1)

- Branch: `use-conversation-names-for-markdown-files`, base `main`; PR #1; commits `e237ada..9f361dc`
- Status: merged
- Started: 2026-09-24T13:37:44Z
- Last commit: 2026-09-24T13:57:16Z
- Merged: 2026-09-24T14:13:05Z
- Worked on by: Byron Jones; GitHub Copilot Autofix (co-author trailer on
  `1abbe10`, `aa07810`, and `9f361dc`); other AI assistance not recorded
- What it did: named Markdown files after the conversation title instead of a
  hash plus index; uniqueness refactor of the name generation; updated the
  testing guidance and the capitalization of `Agents.md` in the project's agent
  files.

### Initial development (direct commits to main)

- Commits `a9230ce`, `8977ead`, `d789bfa`, `c50ad88`, directly on `main`
- Status: merged
- Committed: 2026-06-29T20:15:21Z to 2026-06-29T21:00:41Z
- Worked on by: Byron Jones (locally and through the GitHub web interface); AI assistance not recorded
- What it did: created the converter (JSON export to Markdown, per-folder
  output, attachment extraction), its README, `ENGINEERING.md`, `Makefile`, and
  first tests. The repository was then named `AIChatJsonMD` (later renamed).
