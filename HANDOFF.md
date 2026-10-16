# Handoff

Outstanding work and the context needed to resume it. Read after
[ENGINEERING.md](ENGINEERING.md) (the specification). Finished work is recorded
in [CHANGELOG.md](CHANGELOG.md), not here. Update this file in the pull request
that finishes the work, describing the state after it merges, and whenever work
is paused or reprioritized. It holds no commit ids, pull request numbers, or
merge times (`make changelog/status` prints those from git). Owner
abbreviations: Byron means Byron Jones; Agent means the AI agent working in the
session.

Last updated: 2026-10-16, by Claude Sonnet 5.5 via Claude Code.

## 1. Status

- Build, tests, lint (including `shellcheck`), vulnerability scan, and Markdown
  lint pass (`make all`, `make vuln`), and CI passes on the default branch.
- No work is in flight except the unpublished global standard edit (section 3).
- Statement coverage is about 90%. The program has no known failing behavior.

## 2. Resume here

1. `cd ~/dev/AIProJsonMD && git switch main && git pull`.
2. Run `make all` and confirm it passes (it includes `spec_test.go` and the
   shell tests).
3. See what recent work was done and when with `make changelog/status`, and read
   the newest entries in `CHANGELOG.md`.
4. Take the highest-priority open item in section 4. To publish a branch, ask
   Byron to push it, then open the PR with
   `SSL_CERT_FILE="" CGO_ENABLED=1 gh pr create ...` (see section 7).

## 3. In progress

| Item | State | What is left |
| --- | --- | --- |
| Global standard `~/.claude/agents/coding.md` | Edited in Byron's working tree, uncommitted, in the `ByronScottJones/agents` repository. Changes: consistent `AGENTS.md` spelling; the "Project Specification, Handoff, and Changelog" section; and the rule that changelog entries and the handoff hold no facts that exist only after a pull request is opened or merged (they come from git). | Byron reviews (`git -C ~/.claude/agents diff`) and publishes through his usual pull request flow. The project copy `agents/coding.md` must stay byte-identical to it. |

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

### H-08: Shell formatting and tests (owner: Agent)

- Why: the shell standard names `shfmt` (formatter) and `bats-core` (tests).
  Neither is installed on the development machine, so `scripts/changelog-status.sh`
  follows `shfmt` defaults by hand and is tested through a Go test run by
  `make test`. `shellcheck` is enforced by `make lint/shell` and in CI.
- Acceptance: install `shfmt`, add a `fmt/check` step for shell scripts to the
  Makefile and CI, and decide whether to move the script tests to `bats`.

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
- Do not write the PR number, merge time, merged status, or commit ids into
  `CHANGELOG.md` entries or this file. Twice, a merge outran a follow-up commit
  that recorded them and left `main` wrong. Write the entry and update this file
  in the branch, in the PR that does the work, and read the git-derived facts
  with `make changelog/status`.
- A stacked pull request retargets to `main` only when its base branch is
  deleted. Merge the base PR, delete its branch, and confirm the next PR's base
  is `main` before merging it, then confirm the change is on `main`
  (`git merge-base --is-ancestor <sha> origin/main`). An earlier stacked PR was
  stranded once by merging it into its still-existing base. Prefer independent
  PRs.
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
- Merged local branches can be deleted with `git branch -d`; the remote has only
  `main`. The earlier session screenshots in `~/.uictl/screenshots/` were
  deleted by Byron.
