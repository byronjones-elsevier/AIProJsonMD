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

## Handover Notes

Read [ENGINEERING.md](ENGINEERING.md) before making changes, and update it in
the same change set when architecture, decisions, verification status, or known
risks change.

## General Behavior

- Be concise and direct.
- Prefer editing existing code over rewriting files wholesale.
- Never add placeholder logic or `TODO` comments unless explicitly requested.
- Keep unrelated refactors out of focused changes.
- When unsure about intent, ask one targeted question rather than guessing.
