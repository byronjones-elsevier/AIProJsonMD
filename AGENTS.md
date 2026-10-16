# Agent Guidance

This project follows the topic guidance in the `agents/` folder. Load the
matching topic file before making decisions or edits in that area.

## Topic Routing

| Topic | File to load |
| --- | --- |
| Source code, scripts, or general coding practice | [agents/coding.md](agents/coding.md) |
| Git repositories, branches, commits, or pull requests | [agents/git.md](agents/git.md) |
| Go | [agents/go.md](agents/go.md) |
| Test strategy or test implementation | [agents/testing.md](agents/testing.md) |
| Markdown files or documentation authoring | [agents/markdown.md](agents/markdown.md) |
| GNU Makefiles and task runners | [agents/makefile.md](agents/makefile.md) |

## Specification, Handoff, and Changelog

Three living documents describe the project. Their rules are in
[agents/coding.md](agents/coding.md).

| File | Role |
| --- | --- |
| [ENGINEERING.md](ENGINEERING.md) | Primary specification; the source of truth for behavior and design. |
| [HANDOFF.md](HANDOFF.md) | Outstanding work and how to resume it. |
| [CHANGELOG.md](CHANGELOG.md) | Every branch and pull request: what it did, when, and by whom. |

- At the start of a session read `ENGINEERING.md`, then `HANDOFF.md`, then the
  newest `CHANGELOG.md` entries.
- Change behavior in `ENGINEERING.md` in the same change set as the code.
  `spec_test.go` fails if the specification's literal templates, script, or
  versions drift from the code.
- Write the `CHANGELOG.md` entry and update `HANDOFF.md` in the branch, in the
  same pull request as the work. They hold no pull request numbers, merge
  times, or commit ids: run `make changelog/status` to read those from git.

## General Behavior

- Be concise and direct.
- Prefer editing existing code over rewriting files wholesale.
- Never add placeholder logic or `TODO` comments unless explicitly requested.
- Keep unrelated refactors out of focused changes.
- When unsure about intent, ask one targeted question rather than guessing.
