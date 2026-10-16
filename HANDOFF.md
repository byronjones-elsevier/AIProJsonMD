# Handoff

Outstanding work and the context needed to resume it. Read after
[ENGINEERING.md](ENGINEERING.md) (the specification). Finished work is recorded
in [CHANGELOG.md](CHANGELOG.md), not here. Update this file at the end of every
session and whenever work is paused or reprioritized. Owner abbreviations:
Byron means Byron Jones; Agent means the AI agent working in the session.

Last updated: 2026-10-16 (UTC), by Claude Sonnet 5.5 via Claude Code.

## 1. Status

- Default branch `main` is at `590c032` (PR #19 merged). Build, tests, lint,
  vulnerability scan, and Markdown lint pass locally (`make all`, `make vuln`),
  and CI passed on `main`.
- Work in progress: branch `docs/spec-handoff-changelog` (section 3). It has
  uncommitted changes in this checkout and is not pushed.
- Statement coverage is about 89.8%. The program has no known failing behavior.

## 2. Resume here

1. `cd ~/dev/AIProJsonMD && git switch docs/spec-handoff-changelog && git status`.
2. Run `make all` and confirm it passes (it includes `spec_test.go`).
3. Finish the branch (section 3), then ask Byron to push it:
   `git push -u origin docs/spec-handoff-changelog`, and open the PR against
   `main` with `SSL_CERT_FILE="" CGO_ENABLED=1 gh pr create ...` (see section 7).
4. After that, take the highest-priority open item in section 4.

## 3. In progress

| Item | State | What is left |
| --- | --- | --- |
| Branch `docs/spec-handoff-changelog` | Uncommitted: rewritten `ENGINEERING.md`, new `HANDOFF.md`, `CHANGELOG.md`, `spec_test.go`, `run_test.go`, updates to `AGENTS.md`, `README.md`, `agents/coding.md`. No PR. | Act on the round 2 independent specification review (a fresh AI reviewer given only `ENGINEERING.md`), update the `CHANGELOG.md` entry with the final commit SHAs and timestamps, lint, commit, and hand the push to Byron. |
| Global standard `~/.claude/agents/coding.md` | Edited in Byron's working tree, uncommitted, in the `ByronScottJones/agents` repository. Contains two changes: consistent `AGENTS.md` spelling, and the new "Project Specification, Handoff, and Changelog" section. | Byron reviews (`git -C ~/.claude/agents diff`) and publishes through his usual pull request flow. The project copy `agents/coding.md` must stay byte-identical to it. |

## 4. Outstanding work

Priority order. IDs are stable; remove an item when it is done and record it in
the changelog.

### H-01: Confirm the HTML pages in more browsers (owner: Byron or Agent)

- Why: the pages and copy buttons were exercised only in Chrome, from disk, in
  light mode.
- Acceptance: view a conversation page, a folder page, and the main index in
  Safari and Firefox; check dark mode, a table, an inline image, a very long
  breadcrumb, a page with scripts disabled, keyboard activation of Copy, the
  "Failed" state, and a screen reader announcing the status; confirm which
  copy path (Clipboard API or the `execCommand` fallback) ran. Record results
  in the changelog and fix what is wrong. Relates to REQ-HTML-9, REQ-COPY-1 to 4.
- Blocked by: nothing. Agent can drive Chrome through the `uictl` tools; other
  browsers need Byron or a configured tool.

### H-02: Windows (owner: Agent, needs a Windows runner)

- Why: names are made Windows-safe (D-09) and unit tested, but the program and
  tests have never run on Windows, and CI has no Windows job (D-22). The
  permission test is skipped on Windows.
- Acceptance: add `windows-latest` to the CI test matrix and make it pass, or
  record why a test cannot run there; confirm output paths and links on Windows.

### H-03: Reserve the folder name `Unfiled` (owner: Byron decides, Agent implements)

- Why: an export folder literally named `Unfiled` shares the `Unfiled`
  directory with unfiled conversations (specification 5.2). File names stay
  unique, but the two groups merge into one folder page.
- Acceptance: either keep and document (already documented), or reserve the
  name in the folder name set so such a folder becomes `Unfiled_2`. Add a test
  and update section 5.2, REQ-NAME-6, and a decision.

### H-04: Refreshing index pages needs `-overwrite` (owner: Byron decides)

- Why: index pages follow the same keep-existing rule as other files, so
  re-running into an existing directory with a changed export leaves stale
  index and folder pages unless `-overwrite` is set (specification 5.10, D-03).
  Conversation names can also shift when the export order changes.
- Options: keep as is; always regenerate index pages (they are derived, so
  replacing them loses nothing); or add a flag. See "Decisions needed".

### H-05: `govulncheck` in CI can fail an unrelated pull request (owner: Byron decides)

- Why: the vulnerability job runs on every pull request and push, so a newly
  published advisory fails unrelated work until the toolchain or tool version is
  raised.
- Options: leave it (it surfaces real issues quickly); run it on a schedule and
  on `go.mod` changes only; or allow failure with a visible warning.

### H-06: Minor unverified items (owner: Agent)

- The 200-byte name cap has only been unit tested; no real name is near the limit.
- `make install` has not been run.
- No end-to-end test exercises a real, large export in CI; the realistic-data
  procedure (specification 11.2) is manual and uses private data.

### H-07: Remaining specification gaps (owner: Agent)

- Why: two isolated reviews rated the specification "mostly" sufficient to
  rebuild the program. What they still found not fully pinned down:
  byte-exact HTML for every input (the rendering depends on the pinned goldmark
  version; the spec gives the options and 10 executable examples); the full
  contents of `Makefile`, `.golangci.yml`, `.markdownlint.json`, and `ci.yml`
  (described, not quoted); Go-specific behaviors an implementer in another
  language would need to reproduce (named in the spec, not re-derived); and the
  unreachable error messages in section 8, which no test triggers.
- Acceptance: run another isolated review after the next behavior change, answer
  each real gap in `ENGINEERING.md`, and add a test where the spec quotes the
  code. Stop when a round finds only byte-level formatting differences.

## 5. Decisions needed

| Question | Options | Recommendation |
| --- | --- | --- |
| H-03: reserve `Unfiled`? | Keep and document; reserve the name. | Reserve it: it removes a surprising merge for the cost of one set entry. |
| H-04: how should index pages refresh? | Keep as is; always regenerate index pages; add a flag. | Always regenerate the index and folder pages, since they are derived from the whole run and replacing them destroys nothing; keep `-overwrite` for conversation and attachment files. |
| H-05: where should `govulncheck` run? | Every PR; schedule plus `go.mod` changes; warn only. | Keep it on every PR while the project is small; revisit if it blocks unrelated work. |

## 6. Known issues, risks, and unverified areas

- Chat content is written verbatim into the Markdown files; only the HTML
  output escapes it (specification section 7).
- `golangci-lint` must be built with a Go release at least as new as the
  project toolchain; an older build fails type-checking.
- Conversation file names follow the export order and titles, so a re-run into
  an existing directory without `-overwrite` can leave old and new names side by
  side or keep a file for a different conversation.
- The real export used for validation is private (`in_json/`, ignored by git).
  Validation results are recorded as counts only.

## 7. Environment and access notes

- Byron pushes every branch himself from a normal terminal; git over HTTPS and
  `git push` do not work from an agent session (the proxy rejects the tunnel).
  `git fetch origin` works from a session only with the sandbox disabled
  (the remote is SSH). The agent opens pull requests afterwards with
  `SSL_CERT_FILE="" CGO_ENABLED=1 gh pr create ...`, run with the sandbox
  disabled. Verify a push with `gh api repos/<owner>/<repo>/branches/<branch>`.
- A stacked pull request retargets to `main` only when its base branch is
  deleted. Merge the base PR, delete its branch, and confirm the next PR's base
  is `main` before merging it, then confirm the change is on `main`
  (`git merge-base --is-ancestor <sha> origin/main`). PR #18 was stranded once
  by merging it into its still-existing base. Prefer independent PRs.
- If a merge conflict is resolved in the GitHub web editor, re-check the file on
  `main`; one resolution dropped a README sentence and left a lint error.
- Run Go and npm tools inside the agent sandbox with caches in `$TMPDIR`:
  `GOCACHE=$TMPDIR/gocache GOLANGCI_LINT_CACHE=$TMPDIR/glc GOMODCACHE=$TMPDIR/gomod npm_config_cache=$TMPDIR/npm`.
  Network steps (`go get`, `make vuln`) need the sandbox disabled with
  `SSL_CERT_FILE=""`.
- Byron runs destructive commands himself (for example `rm -rf`); the agent
  gives the exact command with the `!` prefix instead of running it.
- The `uictl` tools (`~/dev/uictl-mac-mcp`) let an agent screenshot and click
  Chrome. The first click on an inactive window only activates it.
- Credentials are Byron's `gh` login and SSH keys on his machine. No secret is
  stored in the repository.

## 8. Cleanup debt

These hold chat text from Byron's private export and were created by the agent.
Byron deletes them (`rm -rf` is run by Byron):

- The scratch directory
  `/private/tmp/claude-502/-Users-jonesb7-dev-AIProJsonMD/31cfadf1-3cd9-40a9-a0b8-28d513e0ff42/scratchpad/nav.4T13CV`.
- Screenshots in `~/.uictl/screenshots/` named `20261016T16-22-32.png`,
  `20261016T16-22-43.png`, `20261016T16-22-52.png`, `20261016T16-25-28.png`,
  `20261016T16-37-45.png`, `20261016T16-37-57.png`, `20261016T16-38-08.png`.
- Local branches other than `main` that are merged can be deleted with
  `git branch -d`; the remote has only `main`.
