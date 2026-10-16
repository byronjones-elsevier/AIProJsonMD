# Engineering Specification

- Status: current. This file is the primary specification of `AIProJsonMD`. The
  code, `README.md`, and every other document must agree with it.
- Last reviewed against the code: 2026-10-16, commit `0a37544`.
- Related files: [HANDOFF.md](HANDOFF.md) (outstanding work),
  [CHANGELOG.md](CHANGELOG.md) (history of all work), [README.md](README.md)
  (user guide), and [AGENTS.md](AGENTS.md) (agent guidance).

The key words MUST, MUST NOT, SHOULD, and MAY are used as in RFC 2119.
Requirements have stable IDs such as `REQ-NAME-3`. Rules that need exact detail
are in sections 3 to 5, and the appendices contain the literal stylesheet,
templates, and script. A test (`spec_test.go`) fails if the appendices, the
dependency versions, or the Windows reserved-name list in this file drift from
the code.

## 1. Overview

`AIProJsonMD` is a command-line program that converts one AI Pro chat export
(a JSON file) into a browsable archive: one Markdown file and/or one HTML page
per conversation, grouped into folders that mirror the export's folders, with
every attachment extracted to a local `assets` directory.

Audience: a person who exports their own chat history and wants to read, search,
version, or share it as plain files.

In scope:

- Reading a single export file and writing files to a local directory.
- Markdown output, and self-contained HTML output with an index, folder pages,
  a breadcrumb on each page, and copy buttons on code blocks.
- Safe handling of untrusted chat content.

Non-goals: network access of any kind, editing or deleting existing files,
reading anything other than the export file, converting the archive back to the
export format, and a graphical interface.

## 2. Requirements

### 2.1 Interface

- REQ-CLI-1: The program MUST be one command, `AIProJsonMD`, with the flags in
  section 3.1 and no positional arguments.
- REQ-CLI-2: `-in` is required. Without it the program MUST print the usage text
  and fail with the error `missing required -in flag`.
- REQ-CLI-3: `-assets` MUST be a single directory name (section 3.2).
- REQ-CLI-4: `-format` MUST accept `md`, `html`, or `both`, case-insensitively,
  and default to `md`.
- REQ-CLI-5: `-h` or `-help` MUST print the usage text and exit with status 0.
- REQ-CLI-6: On success the program MUST print the summary in section 3.4 to
  standard output. Warnings and errors MUST go to standard error.

### 2.2 Input

- REQ-IN-1: The program MUST read the export described in section 4.1 and ignore
  fields it does not use.
- REQ-IN-2: An export with no conversations MUST fail with
  `no history entries found`.
- REQ-IN-3: Fields that the export types inconsistently (`fileSize`, `folderId`,
  `fileType`, `imageBase64`) MUST NOT cause a failure. A value that is not a
  string is treated as absent.

### 2.3 Output layout and names

- REQ-OUT-1: Output MUST follow the layout in section 4.2: one directory per
  export folder under `-out`, a directory `Unfiled` for conversations without a
  mapped folder, and an assets directory inside each folder directory whose
  conversations have attachment references.
- REQ-OUT-2: Existing files MUST NOT be replaced unless `-overwrite` is set. The
  program MUST NOT delete anything.
- REQ-OUT-3: Files and directories the program creates MUST have mode `0600` and
  `0700`. A file that already exists and is replaced with `-overwrite` keeps its
  current mode; that is the only exemption.
- REQ-OUT-4: For the same input and options, on the same platform and
  filesystem, the output MUST be byte-for-byte identical on every run. Output
  MUST NOT contain timestamps or random values. Byte parity with a different
  implementation is not required; the observable behavior in this specification
  is.
- REQ-NAME-1: Names derived from the export MUST be sanitized (section 5.3).
- REQ-NAME-2: A sanitized name MUST NOT exceed 200 bytes and MUST remain valid
  UTF-8.
- REQ-NAME-3: Names that are Windows reserved device names MUST be prefixed so
  that output can be copied to Windows.
- REQ-NAME-4: Conversation file names MUST be unique within a folder, ignoring
  case (section 5.4).
- REQ-NAME-5: The conversation name `index` MUST be reserved in every folder.
- REQ-NAME-6: Folder directory names MUST be unique ignoring case. The names
  `index.html` and `Unfiled` (in any capitalization) are reserved: an export
  folder with one of those names MUST get a numeric suffix (section 5.2). The
  `Unfiled` directory holds only conversations without a mapped folder.

### 2.4 Markdown output

- REQ-MD-1: Markdown files MUST have exactly the structure in section 4.3.
- REQ-MD-2: Markdown output MUST NOT depend on whether HTML is also written.

### 2.5 Attachments

- REQ-ATT-1: Attachment references MUST be collected from the conversation and
  from each message, de-duplicated, and extracted once per conversation
  (section 5.5).
- REQ-ATT-2: An attachment's bytes MUST come from `content` if present, otherwise
  from decoded `imageBase64`.
- REQ-ATT-3: Identical content with the same name MUST be reused, and different
  content with the same name MUST get a numeric suffix.
- REQ-ATT-4: An attachment that cannot be extracted MUST NOT stop the run. The
  output MUST record why, and a warning MUST be logged.

### 2.6 HTML output

- REQ-HTML-1: Each conversation page MUST have the structure in section 4.4.
- REQ-HTML-2: Messages MUST be rendered from Markdown with fenced and indented
  code, lists, tables, headings, links, strikethrough, and task lists.
- REQ-HTML-3: A single newline inside a message MUST render as a line break. A
  blank line MUST start a new paragraph.
- REQ-HTML-4: Raw HTML written in a message MUST be shown as escaped text. It
  MUST NOT be executed, passed through, or silently dropped.
- REQ-HTML-5: Links and images whose URL uses `javascript:`, `vbscript:`,
  `file:`, or a `data:` type other than png, gif, jpeg, or webp MUST be rendered
  with an empty URL (`href=""` or `src=""`). The scheme match is case-insensitive.
- REQ-HTML-6: The role shown on a message MUST NOT reach an HTML attribute. A
  fixed class is chosen from the role (section 5.8).
- REQ-HTML-7: Image attachments MUST be shown inline; other attachments MUST be
  links.
- REQ-HTML-8: Pages MUST be self-contained: no external stylesheet, script,
  font, or image, and no network request when opened.
- REQ-HTML-9: Pages MUST follow the system light or dark preference with no
  control on the page.

### 2.7 Navigation

- REQ-NAV-1: HTML output MUST have three levels: the main index, a page per
  folder, and a page per conversation.
- REQ-NAV-2: Every page below the main index MUST show a breadcrumb as its only
  navigation. Its last crumb is the current page and is not a link.
- REQ-NAV-3: All links between pages MUST be relative.
- REQ-NAV-4: The main index MUST list folders alphabetically, ignoring case, with
  `Unfiled` last. Folder pages MUST list conversations in export order.

### 2.8 Copy buttons

- REQ-COPY-1: Every code block MUST get a Copy button, added by script so that a
  page without script shows plain code blocks.
- REQ-COPY-2: Only pages that contain a code block MAY contain a script.
- REQ-COPY-3: A page with the script MUST allow exactly that script through its
  Content-Security-Policy by SHA-256 hash.
- REQ-COPY-4: The button MUST copy the block's exact text, give visible and
  spoken feedback, be a keyboard-reachable `<button>` with an accessible name,
  and be hidden when printing (section 5.9).

### 2.9 Security and privacy

- REQ-SEC-1: All chat-derived text MUST be treated as untrusted and escaped for
  its context.
- REQ-SEC-2: Every page MUST carry a Content-Security-Policy that blocks external
  resources and, except for the script in REQ-COPY-3, all scripts.
- REQ-SEC-3: Output paths MUST stay inside `-out` (no traversal through names,
  folder names, or `-assets`).
- REQ-SEC-4: The program MUST NOT open network connections or send telemetry.
- REQ-SEC-5: Logs MUST NOT contain message text.

### 2.10 Errors

- REQ-ERR-1: Any error other than a per-attachment failure MUST stop the run,
  be logged once by the program to standard error, and exit with status 1. (For
  a flag error the `flag` package first prints its own message and the usage
  text; for a missing `-in` the usage text is printed before the error is
  logged.)
- REQ-ERR-2: Errors MUST say what failed and why, and wrap the cause.

### 2.11 Non-functional

- REQ-NFR-1: The project MUST build with the Go version and toolchain in
  `go.mod` on macOS and Linux.
- REQ-NFR-2: The only third-party runtime dependency is goldmark (section 9).
- REQ-NFR-3: `gofmt`, `go vet`, `golangci-lint`, and `govulncheck` MUST report
  nothing.
- REQ-NFR-4: Tests MUST pass with the race detector. Statement coverage SHOULD
  stay at or above 85%.
- REQ-NFR-5: Every requirement in this specification MUST be verified by a named
  test or a named procedure in section 11.

## 3. External interfaces

### 3.1 Command line

```text
AIProJsonMD -in <export.json> [-out <dir>] [-assets <name>] [-format <md|html|both>] [-overwrite]
```

The Go `flag` package parses the arguments, so `-flag value`, `-flag=value`,
and `--flag` forms are all accepted.

| Flag | Default | Meaning |
| --- | --- | --- |
| `-in` | none, required | Path to the export JSON file. A value that is empty or only whitespace counts as missing. |
| `-out` | `out_markdown` | Output directory. Created, with intermediate directories, if absent. |
| `-assets` | `assets` | Name of the attachments directory inside each folder directory. |
| `-format` | `md` | `md` writes Markdown only. `html` writes HTML pages and indexes only. `both` writes everything. |
| `-overwrite` | `false` | Replace existing Markdown, HTML, index, and identical-content attachment files instead of keeping them. An attachment that already exists with different bytes is never replaced; the new one gets a numeric suffix (5.5). |

Positional arguments are accepted by the parser and ignored. The values of
`-in`, `-out`, and `-assets` are used exactly as given (not trimmed), apart from
the checks in 3.2. `-out` is not validated: an empty value fails when the
directory is created, with `create output directory: <cause>`.

Usage text is the `flag` package default and goes to standard error. Flags are
listed alphabetically. The program prints each description line indented by four
spaces and a tab; the tab is shown here as spaces:

```text
Usage of AIProJsonMD:
  -assets string
        Per-folder assets directory name (default "assets")
  -format string
        Output format: md, html, or both (default "md")
  -in string
        Path to input JSON file
  -out string
        Output directory for generated files (default "out_markdown")
  -overwrite
        Overwrite existing output and attachment files
```

### 3.2 Validation, in this order

1. Parse flags. An unknown flag or a malformed value prints the `flag` package
   message and usage to standard error, then fails (status 1). `-h` and `-help`
   print usage and exit with status 0 without logging an error.
2. `-in` blank: print the usage text, then fail with `missing required -in
   flag` (logged after the usage text).
3. `-assets` invalid: fail with `invalid -assets value: must be a single
   directory name: "<value>"`. Invalid means: empty or only whitespace; exactly
   `.` or `..`; contains `/` or `\`; or contains a control character (a
   character for which Go's `unicode.IsControl` is true, which is a wider set
   than the one replaced in file names by 5.3). The value is shown as given,
   quoted with Go `%q` rules.
4. `-format` invalid: fail with `invalid -format value: must be md, html, or
   both: "<value>"`. The value is trimmed and lower-cased before comparison; the
   error shows it as given, quoted with Go `%q` rules.
5. Read the file, parse the JSON, require at least one conversation, create the
   output directory, convert (section 5.1).

### 3.3 Exit status and streams

| Situation | Exit status | Standard output | Standard error |
| --- | --- | --- | --- |
| Success | 0 | summary (3.4) | zero or more warnings (8) |
| `-h` or `-help` | 0 | nothing | usage |
| Any error | 1 | nothing | the error (8) |

### 3.4 Summary

On success the program prints these lines to standard output, each ending with
a newline. The Markdown and HTML lines appear only if their format was
requested. The last line always appears, even when no attachment was written.

```text
Done. Wrote <M> of <N> markdown files to <out>
Wrote <H> of <N> HTML files to <out> (index.html <written|kept>, <F> of <T> folder index pages written)
Attachments written under per-folder <assets> directories
```

- `N` is the number of conversations. `M` and `H` count files actually written;
  a file that already existed and was kept is not counted.
- The first line printed starts with `Done.` and a space. With `html` only, that
  prefix is on the HTML line, which is then the first line.
- `index.html` is `written` if the main index was written and `kept` if it
  already existed and `-overwrite` was not set.
- `T` is the number of folder pages (folders that have at least one HTML
  conversation) and `F` is how many were written.

### 3.5 Task runner

The `Makefile` provides these targets. `make` with no target prints help.

| Target | Purpose |
| --- | --- |
| `help` | Lists targets using the `##` comment above each. |
| `default` | Alias for `build`. |
| `all` | `fmt/check`, `vet`, `lint`, `test/race`, `build`. |
| `deps` | `go mod download` then `go mod verify`. |
| `fmt` | `go fmt ./...`. |
| `fmt/check` | Fails if `gofmt -l .` lists any file. |
| `vet` | `go vet ./...`. |
| `lint` | `golangci-lint run ./...`. |
| `test` | `deps`, then `go test ./...`. |
| `test/race` | `deps`, then `go test -race ./...`. |
| `test/cover` | Tests with `coverage.out` and a printed total. |
| `vuln` | `go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...`. |
| `build` | `deps`, then `go build -o AIProJsonMD .`. |
| `install` | `deps`, then `go install .`. |
| `run` | Runs the program with `IN` (required), `FORMAT`, `OUT_DIR`, `ASSETS_DIR`. |
| `clean` | Removes the binary and `coverage.out`. |

Variables use `?=` defaults: `APP_NAME=AIProJsonMD`, `GO=go`, `GOFLAGS=`,
`GOVULNCHECK_VERSION=v1.9.0`, `OUT_DIR=out_markdown`, `ASSETS_DIR=assets`,
`FORMAT=md`.

## 4. Data

### 4.1 Input: the export

One JSON object. Field names are matched the way Go's `encoding/json` matches
them, which ignores case. A value of the wrong type for a field (for example a
number for `content`) makes parsing fail with `parse JSON: <cause>`, except for
the leniently typed fields in REQ-IN-3. A `null` or missing `history` counts as
empty. A top-level `null` is accepted and counts as an empty export (`no history
entries found`); any other top-level value that is not an object fails to parse.
As in Go's `encoding/json`: a byte-order mark or trailing data that is not white
space fails to parse; invalid UTF-8 in a string is replaced by U+FFFD; and if an
object repeats a key, the last one wins. The wording of the cause is not part of
the contract.

| Path | Type | Notes |
| --- | --- | --- |
| `version` | number | Read, not used. |
| `history` | array of conversation | Required to be non-empty. |
| `history[].id` | string | May be empty. Trimmed when used. |
| `history[].name` | string | The conversation title. May be empty. |
| `history[].folderId` | string or `null` | Only a string is used. `null`, a number, or anything else means no folder. |
| `history[].messages` | array of message | |
| `history[].messages[].role` | string | Typically `user` or `assistant`. |
| `history[].messages[].content` | string | Markdown text. |
| `history[].messages[].files` | array of file reference | |
| `history[].files` | array of file reference | Conversation-level references. Often empty even when messages have files. |
| `folders` | array | |
| `folders[].id` | string | A folder with a blank id is ignored. |
| `folders[].name` | string | |
| `folders[].type` | string | Read, not used. |
| `files` | array of file blob | Attachment payloads, keyed by `fileKey`. |
| `files[].fileKey` | string | Blobs with a blank key (after trimming) are ignored. If keys repeat, the last blob wins. |
| `files[].fileName` | string | |
| `files[].fileSize` | string or number | Read, not used. |
| `files[].conversationId` | string | Read, not used. |
| `files[].content` | string | Text payload. If non-empty it is used as is, without trimming. |
| `files[].fileType` | string or other | MIME type such as `image/png`. Only a string is used. |
| `files[].imageBase64` | string or other | Base64 or a `data:` URI. Only a string is used. |

A file reference has `fileKey`, `fileName`, and `fileSize` (unused).

Observed quirks: `fileSize` is a string such as `"7.06 KB"` in some places and a
number in others; many conversations have `folderId: null`; message-level files
can exist while the conversation-level `files` array is empty; the base64 value
can be a `data:` URI, contain line breaks, or lack padding.

### 4.2 Output layout

```text
<out>/
  index.html                      HTML formats only: the main index
  <Folder>/                       one directory per export folder, and Unfiled
    index.html                    HTML formats only: this folder's page
    <Conversation>.md             Markdown formats
    <Conversation>.html           HTML formats
    <assets>/                     only if a conversation in the folder has attachment references
      <attachment files>
```

- Folder directories are named by section 5.2, conversation files by section
  5.4, and attachments by section 5.5.
- The `.md` and `.html` file for a conversation share a base name.
- A folder directory is created only when a conversation is written into it, so
  an export folder with no conversations produces no directory.
- An assets directory is created whenever a conversation has at least one
  attachment reference, even if every extraction then fails, so it can be empty.

### 4.3 Markdown document

For each conversation, with `<title>`, `<id>`, `<role>`, `<content>`:

```markdown
# <title>

_id: `<id>`_

## Messages

### <role>

<content>

### <role>

<content>

## Attachments

- **<name>**: [<file>](<relative-link>)
- **<name>**: _<failure>_
```

- `<title>` is the trimmed conversation name, or `Conversation <n>` (n is the
  1-based position in `history`) if it is blank.
- The `_id:` line, and the blank line after it, appear only if the trimmed id is
  not empty.
- `<role>` is the trimmed role, or `message` if blank. `<content>` is the
  trimmed content. Each message block ends with one blank line.
- The `## Attachments` section appears only if the conversation has at least
  one attachment reference. Each attachment is one list item, either a link
  (`<file>` is the stored file name and `<relative-link>` is the escaped path
  from section 5.7) or the failure text from section 5.5.
- Line endings are `\n`. With no attachments the file ends with the last
  message's content followed by a blank line (two `\n`). With attachments it
  ends with one `\n` after the last list item.
- A conversation with no messages still has `## Messages` followed by a blank
  line. An empty message content gives the role heading, then a blank line, then
  nothing, then a blank line.
- Titles, ids, roles, content, and attachment names are inserted as they are.
  Markdown metacharacters in them are not escaped, except that `[` and `]` in
  link text are escaped (5.7).
- The displayed attachment name is the trimmed `fileName`, or `attachment_<k>`
  (`k` is the 1-based position in the collected list) if blank. It is not
  changed by sanitizing or by appending an extension; the link text is the
  stored file name.

### 4.4 HTML pages

All pages share the head in Appendix B (the Content-Security-Policy differs
only as described in section 5.9) and the stylesheet in Appendix A. They have
no external references. Every HTML file ends with `</html>` and one newline.

`<title>` is the conversation title on a conversation page, the folder name on a
folder page, and `AI Pro Export` on the main index. The "folder name" shown on
HTML pages (breadcrumb, folder page `<title>` and `<h1>`, main index `<h2>`, and
the main index sort key) is the folder's directory name from section 5.2, not
the name in the export.

Conversation page (`<Conversation>.html`), in document order:

1. Breadcrumb (`nav.breadcrumb`): `Main index` (link to the main index), the
   folder directory name (link to the folder page), and the conversation title
   as the current page. The first link is the relative path to the main index
   (`../index.html` from a page inside a folder directory) and the folder link
   is `index.html`.
2. `<h1>` with the title.
3. If the id is non-empty: `<p class="meta">id: <code>id</code></p>`.
4. `<h2>Messages</h2>`, then one `<section class="message <role-class>">` per
   message with an `<h3>` holding the role and a `<div class="body">` holding
   the rendered Markdown.
5. If there are attachments: `<h2>Attachments</h2>` and `<ul class="attachments">`
   with one `<li>` per attachment: the original name in `<strong>`, then either
   the failure text in `<em>` or a link to the file. An image attachment is shown
   as an inline `<img>`, a `<br>`, and then the stored file name, all inside the
   link.
6. If the page has a code block: the script in Appendix C, after `</main>`.

Folder page (`<Folder>/index.html`): breadcrumb `Main index` then the folder
name as the current page; `<h1>` with the folder name; a `<ul>` with one link per
conversation in export order, targeting the conversation's HTML file.

Main index (`index.html`): no breadcrumb; `<h1>AI Pro Export`; for each folder
an `<h2>` whose text links to the folder page, followed by a `<ul>` of links to
that folder's conversations, in export order. Folders are sorted alphabetically
ignoring case, with `Unfiled` last.

Template data (the names used in Appendix B): a conversation page has `Title`,
`ID`, `CSP`, `Script` (empty unless a code block page), `Nav.Crumbs` (each with
`Label` and `Href`, where `Href` is empty for the current page), `Messages`
(each with `Role`, `RoleClass`, and `Body`, the rendered HTML), and
`Attachments` (each with `Name`, `Href`, `LinkText`, `IsImage`, and `Failure`).
A folder page has `Title`, `CSP`, `Nav`, and `Conversations` (each with `Title`
and `Href`). The main index has `Title`, `CSP`, and `Folders` (each with `Name`,
`Href`, and `Conversations`, each with `Title` and `Href`).

The literal templates are in Appendix B. Elements and class names there are
normative; whitespace inside the page body is not significant to a reader but
the output MUST be deterministic (REQ-OUT-4).

### 4.5 Worked example

Input `export.json`:

```json
{
  "version": 1,
  "folders": [
    { "id": "f1", "name": "Project A", "type": "chat" }
  ],
  "history": [
    {
      "id": "c1",
      "name": "Deploy: notes",
      "folderId": "f1",
      "messages": [
        { "role": "user", "content": "Run this:\nthen that\n\n<b>not bold</b>" },
        {
          "role": "assistant",
          "content": "```sh\necho \"hi\"\n```",
          "files": [ { "fileKey": "k1", "fileName": "run (1).sh" } ]
        }
      ]
    },
    {
      "id": "c2",
      "name": "Deploy: notes",
      "folderId": "f1",
      "messages": [
        { "role": "user", "content": "Second chat with the same title", "files": [ { "fileKey": "gone", "fileName": "lost.txt" } ] }
      ]
    },
    {
      "id": "c3",
      "name": "",
      "folderId": null,
      "messages": [ { "role": "user", "content": "No title, no folder" } ]
    }
  ],
  "files": [
    { "fileKey": "k1", "fileName": "run (1).sh", "content": "echo hi\n" }
  ]
}
```

Command and output (the warning goes to standard error):

```text
$ AIProJsonMD -in export.json -out out -format both
WARN attachment skipped name=lost.txt reason="missing content" fileKey=gone
Done. Wrote 3 of 3 markdown files to out
Wrote 3 of 3 HTML files to out (index.html written, 2 of 2 folder index pages written)
Attachments written under per-folder assets directories
```

Files written:

```text
out/index.html
out/Project A/index.html
out/Project A/Deploy_ notes.md
out/Project A/Deploy_ notes.html
out/Project A/Deploy_ notes_2.md
out/Project A/Deploy_ notes_2.html
out/Project A/assets/run (1).sh
out/Unfiled/index.html
out/Unfiled/7c1c97df17_3.md
out/Unfiled/7c1c97df17_3.html
```

The warning is shown without the date and time prefix that the logger adds.
Notes: `:` in the title became `_`; the second conversation with the same title
got `_2`; the untitled conversation is named from the first ten hex characters
of the SHA-256 of its id plus its position (`_3`); `lost.txt` has no blob, so
it produced a warning and a failure note but no file.

`out/Project A/Deploy_ notes.md`:

````markdown
# Deploy: notes

_id: `c1`_

## Messages

### user

Run this:
then that

<b>not bold</b>

### assistant

```sh
echo "hi"
```

## Attachments

- **run (1).sh**: [run (1).sh](assets/run%20%281%29.sh)
````

The body of `out/Project A/Deploy_ notes.html` after the head (the stylesheet
and script are elided here; see the appendices). Note the line break, the
escaped `<b>`, the code block, and the escaped attachment link:

```html
<nav class="breadcrumb" aria-label="Breadcrumb">
<ol>
<li><a href="../index.html">Main index</a></li>
<li><a href="index.html">Project A</a></li>
<li aria-current="page">Deploy: notes</li>
</ol>
</nav>
<h1>Deploy: notes</h1>
<p class="meta">id: <code>c1</code></p>
<h2>Messages</h2>
<section class="message role-user">
<h3>user</h3>
<div class="body">
<p>Run this:<br>
then that</p>
<p>&lt;b&gt;not bold&lt;/b&gt;</p>
</div>
</section>
<section class="message role-assistant">
<h3>assistant</h3>
<div class="body">
<pre><code class="language-sh">echo &quot;hi&quot;
</code></pre>
</div>
</section>
<h2>Attachments</h2>
<ul class="attachments">
<li><strong>run (1).sh</strong>: <a href="assets/run%20%281%29.sh">run (1).sh</a></li>
</ul>
```

The Content-Security-Policy of that page, decoded (it has a code block):

```text
default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; base-uri 'none'; script-src 'sha256-kni1OruwCTgnOK5TRYkWn53EbeaEeT/kYJSDaLMPjuA='
```

`out/Project A/Deploy_ notes_2.html` has no code block, so its policy is the
base policy only and it contains no script. `out/Project A/index.html` and
`out/index.html` are:

```html
<h1>Project A</h1>
<ul>
<li><a href="Deploy_%20notes.html">Deploy: notes</a></li>
<li><a href="Deploy_%20notes_2.html">Deploy: notes</a></li>
</ul>
```

```html
<h1>AI Pro Export</h1>
<h2><a href="Project%20A/index.html">Project A</a></h2>
<ul>
<li><a href="Project%20A/Deploy_%20notes.html">Deploy: notes</a></li>
<li><a href="Project%20A/Deploy_%20notes_2.html">Deploy: notes</a></li>
</ul>
<h2><a href="Unfiled/index.html">Unfiled</a></h2>
<ul>
<li><a href="Unfiled/7c1c97df17_3.html">Conversation 3</a></li>
</ul>
```

(The `<h1>` blocks above are the part of those pages after the head and
breadcrumb; the folder page also has its breadcrumb before the `<h1>`.)

## 5. Behavior and algorithms

### 5.1 Run sequence

1. Validate arguments (section 3.2).
2. Read `-in` as bytes and parse it as JSON into the export model. Failure:
   `read input: <cause>` or `parse JSON: <cause>`.
3. If `history` is empty, fail with `no history entries found`.
4. Create `-out` and parents with mode `0700`. Failure:
   `create output directory: <cause>`.
5. Build the converter: index blobs by trimmed `fileKey`; map folder ids to
   directories (5.2); if HTML is requested, parse the templates and build the
   Markdown renderer. Template parse failures are fatal.
6. For each conversation, in `history` order, with 0-based position `i`, run
   section 5.6. An error stops the run: `conversation <i>: <cause>`.
7. If HTML is requested, write the main index and the folder pages (5.10).
8. Print the summary (3.4).

### 5.2 Folder directories

For each export folder, in `folders` order, skipping blank ids:

1. `name` = `sanitize(folder.name)` (5.3); if that is empty, use `folder`.
2. Keep a set of used names, compared lower-cased. It starts containing
   `index.html` and `Unfiled`.
3. `pathName` = `name`. While the lower-cased `pathName` is in the set, set
   `pathName` to `<name>_<n>` for n = 2, 3, and so on. Add the final lower-cased
   `pathName` to the set.
4. The folder's directory is `<out>/<pathName>`.

Folder ids are trimmed when mapped. If two folders share an id, the later one
wins for lookups (the earlier one still consumed a directory name). An export
folder that has no conversations is not created and does not appear on any page.

A conversation's folder is found from `folderId` if it is a string (trimmed)
that matches a mapped folder id. Otherwise the directory is `<out>/Unfiled`.
Because `Unfiled` is in the starting set, an export folder named `Unfiled` in any
capitalization is renamed by step 3 to the first free suffixed name (`Unfiled_2`,
`unfiled_3`, and so on), so `<out>/Unfiled` holds only conversations that have
no mapped folder.

On the main index, "`Unfiled` last" is an exact, case-sensitive match on the
directory name `Unfiled`. A renamed folder such as `Unfiled_2` is sorted
alphabetically with the others. Directory names are unique ignoring case, so no
two groups have the same lower-cased name and the order is fully determined.

### 5.3 Sanitizing a name

`sanitize(s)` for folder names, conversation names, and attachment names:

1. Trim leading and trailing whitespace. If nothing remains, return the empty
   string.
2. Replace each character in `< > : " / \ | ? *` and each control character
   U+0000 to U+001F with `_` (the regular expression is
   `[<>:"/\\|?*\x00-\x1F]`). Only U+0000 to U+001F are treated as control
   characters here; DEL and the C1 controls are kept.
3. Remove all leading `.`, then remove all trailing `.` and spaces.
4. Take the text before the first `.`, remove trailing spaces, and upper-case it.
   If it is one of these Windows reserved names, prefix the name with `_`:
   `CON`, `PRN`, `AUX`, `NUL`, `COM1` to `COM9`, or `LPT1` to `LPT9`.
5. If the name is longer than 200 bytes, shorten it: let `ext` be the text from
   the last `.`, or empty if it is longer than 16 bytes or not shorter than the
   limit; cut the rest at `200 - len(ext)` bytes, moving back to a UTF-8
   character boundary; remove trailing `.` and spaces from the cut text; append
   `ext`.

| Input | Result |
| --- | --- |
| `My / Chat: Notes? v1.2` | `My _ Chat_ Notes_ v1.2` |
| `..hidden..` | `hidden` |
| `a .` | `a` |
| `CON` | `_CON` |
| `aux.txt` | `_aux.txt` |
| `COM0` | `COM0` |
| `console.txt` | `console.txt` |
| 500 letters followed by `.txt` | 196 letters followed by `.txt` |

### 5.4 Conversation file names

For the conversation at 0-based position `i` in directory `folderDir`:

1. `base` = `sanitize(name)`.
2. If `base` is empty: `base` = first 10 hex characters of the SHA-256 of the
   trimmed id (of the empty string if there is no id), then `_`, then `i + 1`.
3. If `base` ends in `.md` (any case), remove that extension. If `base` is now
   empty, `base` = `conversation_<i + 1>`. (Step 1 removes leading dots, so
   `base` cannot be empty here today; the rule is a guard that cannot trigger.)
4. Names are tracked per directory, compared lower-cased. The name `index` is
   marked used in every directory before any conversation is named.
5. If `base` is unused, use it. Otherwise try `<base>_2`, `<base>_3`, and so on
   (the suffix keeps the original capitalization of `base`) and use the first
   unused. Mark the chosen name used.
6. The Markdown file is `<folderDir>/<name>.md` and the HTML file is
   `<folderDir>/<name>.html`. Both formats use the same `<name>`.

Example: in one folder, conversations titled `index`, `Index`, and `Notes`
become `index_2`, `Index_3`, and `Notes`.

### 5.5 Attachments

Per conversation, once, before any file for it is written:

1. Collect references: the conversation's `files`, then each message's `files`,
   in order. Drop a reference whose trimmed `fileKey` and trimmed `fileName`
   pair has already been collected. Keep the list index `k` (0-based).
2. If the list is empty, there is no assets directory and no Attachments
   section. Otherwise create `<folderDir>/<assets>` (mode `0700`).
3. For each reference: `name` is the trimmed `fileName`, or `attachment_<k+1>`
   if blank. Look up the blob by trimmed `fileKey`.
   - No blob: the failure text is "missing content for fileKey", a space, then
     the key wrapped in backticks, and a warning is logged.
   - Otherwise get the payload (below). On error: failure
     `failed to load attachment payload: <cause>`.
   - `safe` = `sanitize(resolvedName)`, or `attachment_<k+1>` if empty.
   - Choose the path (below). On error: failure
     `failed to resolve attachment path: <cause>`.
   - Write the file with the rule in 5.6. On error: failure
     `failed to write attachment: <cause>`.
   - Success: the attachment is at `<assets>/<stored file name>`.
4. Every failure is also logged as a warning (section 8).

Payload: if `content` is non-empty, the payload is its UTF-8 bytes and the name
is unchanged. Otherwise if `imageBase64` is not a non-empty string, the error is
`no attachment payload found`. Otherwise decode it: trim whitespace (an empty
result is the error `empty base64 payload`); if it starts with `data:` and has
a comma, drop everything through the first comma; delete `\n` and `\r`; decode as
standard base64, and if that fails, as unpadded base64; if both fail, the error
is `decode imageBase64: <both causes>`. If the name has no extension, append one:
from `fileType` using the map below if it is a string that appears in the map,
else from the content type detected from the decoded bytes (the standard
library's content sniffing), else nothing. A `fileType` that is a string but not
in the map falls through to sniffing. A sniffed type can carry parameters (for
example `text/plain; charset=utf-8`) and is compared as is, so only the six image
types in the map can match. The type in a `data:` header is ignored. A name has
an extension if the text from its last `.` is non-empty, so `.bashrc` and `name.`
have one. This test happens before sanitizing, so `name.` counts as having an
extension and is then stored as `name`.

Content sniffing recognizes these image signatures at the start of the bytes:
PNG (`\x89PNG\r\n\x1a\n`), JPEG (`\xFF\xD8\xFF`), GIF (`GIF87a` or `GIF89a`),
WebP (`RIFF`, any four bytes, then `WEBPVP`), and BMP (`BM`). Any other bytes,
including SVG text, give no extension.

| MIME type (case-insensitive, trimmed) | Extension |
| --- | --- |
| `image/png` | `.png` |
| `image/jpeg`, `image/jpg` | `.jpg` |
| `image/gif` | `.gif` |
| `image/webp` | `.webp` |
| `image/bmp` | `.bmp` |
| `image/svg+xml` | `.svg` |

Choosing the path: split `safe` into `base` and `ext` (the last `.` and after).
For attempt 0 use `safe`; for attempt `a >= 1` use `<base>_<a + 1><ext>`. For
each candidate in `<folderDir>/<assets>`: if it exists and its bytes equal the
payload, use that path (the write in 5.6 then keeps it without `-overwrite` and
rewrites the same bytes with `-overwrite`); if it exists with different bytes,
try the next attempt; if it does not exist, use it; any other error is a failure.

### 5.6 Writing files

- Without `-overwrite`: create the file exclusively (`O_EXCL`) with mode
  `0600`. If it already exists, keep it and report "not written". Any other
  failure is an error. Write errors are reported together with any close error.
- With `-overwrite`: write the file with mode `0600`, replacing existing
  contents (an existing file keeps its current mode), and report "written".
- Directories are created with `MkdirAll` and mode `0700`; existing directories
  are not changed.
- Per conversation the order is: create the folder directory; name the files;
  extract attachments; write the Markdown file if requested; render and write the
  HTML file if requested and record its index entry.

### 5.7 Links

`escapeLinkPath(p)` splits a slash-separated relative path on `/`, escapes each
segment with URL path escaping, additionally turns `(` into `%28` and `)` into
`%29`, and joins the segments with `/`. In Markdown, `[text](path)` is written
with `[` and `]` in the text escaped as `\[` and `\]`. Links from a conversation
file to its attachments are `<assets>/<stored file name>`; the displayed text is
the stored file name. Attachment paths are relative to the conversation's own
folder directory.

### 5.8 HTML rendering of messages

Each message's trimmed content is converted with goldmark configured with: the
Table, Strikethrough, and TaskList extensions; hard wraps (a single newline
becomes `<br>`); and goldmark's default safe URL handling (links and images with
`javascript:`, `vbscript:`, `file:`, or a `data:` type other than png, gif,
jpeg, or webp get an empty `href` or `src`, matched case-insensitively).
goldmark's raw-HTML renderers are replaced so that:

- inline raw HTML is written as the escaped source text;
- an HTML block is written as `<pre class="raw-html">` containing the escaped
  source lines (including a closing line, if any), then `</pre>`.

The replacement renderers are registered with a higher priority than goldmark's
defaults, so they take over for those two node kinds. Examples follow in
"Rendering examples" below.

Escaping converts `<`, `>`, `&`, and `"` to entities.

`role-class(role)`: the lower-cased role `user` gives `role-user`, `assistant`
gives `role-assistant`, and anything else gives `role-other`.

All other text on a page (titles, ids, folder names, roles, attachment names,
failure text, link targets) is inserted by an HTML template engine that escapes
for its context. The page template engine MUST be context-aware escaping
(Go `html/template` is used). The entity forms are therefore those of that
engine (for example `&#34;` for a double quote and `&#39;` for an apostrophe in
text, and `&#43;` for `+` in an attribute); a different implementation may use
equivalent entities, since a reader and the tests see the decoded text.

Beyond the options above, goldmark runs with its defaults: HTML output is not
XHTML (`<br>`, not `<br />`), headings get no automatic ids, and a fenced code
block with an info string becomes `<pre><code class="language-<info>">`.
goldmark escapes text with `&lt;`, `&gt;`, `&amp;`, and `&quot;`.

The role shown in the `<h3>` is the trimmed role, or `message` if it is blank.
The class is chosen from that value, lower-cased.

Attachments: an attachment is an inline image if its stored file name has the
extension `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, or `.svg`
(case-insensitive). Its `<img>` has `src` equal to the escaped path, `alt`
equal to the original name, and `loading="lazy"`.

A message is a "code block page" trigger if its rendered HTML contains the text
`<pre><code`. A page is a code block page if any message triggers.

#### Rendering examples

Each pair is a message's content and the HTML produced for it. A raw HTML block
has a newline before `</pre>` when other content follows it, and none when it is
the last thing in the message. `TestSpecRenderingExamples` runs every pair.

Input:

```markdown
Linux: a
Windows: b
```

Output:

```html
<p>Linux: a<br>
Windows: b</p>
```

Input:

```markdown
use <b>bold</b> here
```

Output:

```html
<p>use &lt;b&gt;bold&lt;/b&gt; here</p>
```

Input:

```markdown
<div>
x
</div>
```

Output:

```html
<pre class="raw-html">&lt;div&gt;
x
&lt;/div&gt;</pre>
```

Input:

```markdown
before

<div>
x
</div>

after
```

Output:

```html
<p>before</p>
<pre class="raw-html">&lt;div&gt;
x
&lt;/div&gt;
</pre>
<p>after</p>
```

Input:

```markdown
[x](javascript:alert(1))
```

Output:

```html
<p><a href="">x</a></p>
```

Input:

```markdown
[x](JaVaScRiPt:alert(1))
```

Output:

```html
<p><a href="">x</a></p>
```

Input:

```markdown
![x](file:///etc/passwd)
```

Output:

```html
<p><img src="" alt="x"></p>
```

Input:

```markdown
[x](https://example.com/a?b=1&c=2)
```

Output:

```html
<p><a href="https://example.com/a?b=1&amp;c=2">x</a></p>
```

Input:

````markdown
```go
fmt.Println(1)
```
````

Output:

```html
<pre><code class="language-go">fmt.Println(1)
</code></pre>
```

Input:

```markdown
~~gone~~ and **bold**
```

Output:

```html
<p><del>gone</del> and <strong>bold</strong></p>
```

### 5.9 Content-Security-Policy and the copy script

The base policy is:

```text
default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; base-uri 'none'
```

It is the policy of the main index, folder pages, and conversation pages with
no code block. A code block page appends
`; script-src 'sha256-<base64 SHA-256 of the script text>'` where the script text
is exactly the bytes between `<script>` and `</script>` in the page: the script
in Appendix C with no leading or trailing whitespace, inserted as trusted
JavaScript (not quoted or escaped). The policy
is placed in a `<meta http-equiv="Content-Security-Policy">` tag; the template
engine escapes quotes in the attribute, which a browser decodes. No other
script, and no `unsafe-inline` or `unsafe-eval` for scripts, is ever allowed.

The script (Appendix C) behaves as follows when the page is opened:

1. It finds every `pre > code`. For each, it wraps the `<pre>` in
   `<div class="code-block">` and appends a `<button type="button"
   class="copy-button">` with the text `Copy` and `aria-label` `Copy code to
   clipboard`, and a `<span class="sr-only" role="status">`.
2. On click it copies the code element's `textContent`. If the Clipboard API is
   available and the page is a secure context it uses
   `navigator.clipboard.writeText`, falling back to a hidden read-only
   `<textarea>` and `document.execCommand("copy")` if that fails or is
   unavailable.
3. On success the button text becomes `Copied!` and the status region reads
   `Code copied to clipboard`; on failure the button reads `Failed` and the
   region `Copy failed`. After 2000 ms both return to `Copy` and empty. A new
   click restarts the timer.
4. The stylesheet positions the button at the top right of the block, shows it
   with reduced opacity until hover or keyboard focus, and hides it when
   printing.

The script text MUST NOT contain `</`, `<!--`, `-->`, `/*`, or `//`, so that the
template engine inserts it unchanged and the hash stays valid.

### 5.10 Index pages

- Every converted HTML conversation records an entry, whether its file was
  written or kept: its folder directory name, its title, the escaped path from
  `-out`, and its HTML file name.
- The main index is written at `<out>/index.html`. Its folder list is sorted by
  lower-cased folder name, with `Unfiled` last. Each folder heading links to
  `<folder>/index.html` (escaped), and its conversations are in export order.
- A folder page is written at `<folderDir>/index.html` for every folder that has
  at least one HTML conversation. Folders are processed in order of first
  appearance. The page links to the main index with a relative path.
- Index pages are written with the rule in 5.6, so existing ones are kept
  unless `-overwrite` is set. They are written even if every conversation file
  already existed.
- With `-format md` no index page or HTML file is written.

### 5.11 Conventions

- Text is UTF-8 without a byte-order mark. The program writes `\n` line
  endings. Message content is copied as it is, so carriage returns in the export
  stay in the Markdown file.
- The whole input file is read into memory. There are no limits on file size or
  conversation count beyond available memory and the filesystem.
- Existence checks use the filesystem, so names that differ only in case behave
  as the platform's filesystem treats them (on a case-insensitive filesystem
  `A.png` and `a.png` are the same file). Determinism (REQ-OUT-4) holds per
  platform.
- Symbolic links in `-out` are followed by the operating system; the program does
  nothing special for them. An existing file at the `-out` path, or `-in` inside
  `-out`, is not detected.
- Trimming removes Unicode white space (Go's `strings.TrimSpace`). The hex digits
  of the id hash are lower-case.
- Conversation positions in error messages (`conversation <i>`) are 0-based;
  positions in names and fallback titles (`Conversation <n>`, `_<n>`) are
  1-based.

## 6. Architecture

`main` calls `run(args, stdout)`, which returns an error. `run` validates,
reads, and drives a `converter`. The converter owns the folder map, the blob
index, the per-directory name tracker, and the list of index entries. Data flows:
export JSON, then export model, then per-conversation (names, attachments), then
the Markdown renderer and the HTML renderer (both read the same attachment list),
then files; finally the index pages.

| File | Purpose |
| --- | --- |
| `main.go` | Flag parsing, validation, the run sequence, the summary, and `main` (logs the error and exits 1). |
| `model.go` | Types for the export JSON. |
| `format.go` | The `-format` value, `parseFormat`, and `ErrInvalidFormat`. |
| `convert.go` | Converter setup, folder directory mapping, conversation file naming, reserved names, and link escaping. |
| `conversation.go` | Per-conversation work: attachment extraction, Markdown rendering, writing Markdown and HTML, and the index pages. |
| `files.go` | Name sanitizing and truncation, Windows reserved names, no-clobber writes, base64 decoding, attachment payloads, and asset path choice. |
| `html.go` | The page, folder, and index templates and styles, the Markdown renderer, raw-HTML escaping, navigation, and role classes. |
| `copy.go` | The copy-button script and the Content-Security-Policy builder. |
| `main_test.go`, `files_test.go`, `convert_test.go`, `html_test.go`, `nav_test.go`, `copy_test.go` | Unit and end-to-end tests (section 11). |
| `spec_test.go` | Fails if this file drifts from the code (section 11). |
| `Makefile` | Task runner (3.5). |
| `.golangci.yml`, `.markdownlint.json` | Linter configuration. |
| `.github/workflows/ci.yml`, `.github/dependabot.yml` | CI and dependency updates. |
| `go.mod`, `go.sum` | Module, pinned versions, and checksums. |
| `README.md`, `AGENTS.md`, `agents/`, `HANDOFF.md`, `CHANGELOG.md`, `SECURITY.md`, `LICENSE` | Documentation and guidance. |
| `in_json/`, `out_markdown/` | Local input and output, ignored by git. |

Error values: `ErrMissingInput`, `ErrNoHistory`, `ErrInvalidFormat`,
`ErrInvalidAssetsDir`, `ErrEmptyBase64`, and `ErrNoPayload` are exported
sentinels that callers can match with `errors.Is`.

## 7. Security and privacy model

Trust boundary: the export file is untrusted data. The program, its templates,
and its script are trusted. The reader of the output is protected from the
export's contents.

| Threat | Mitigation |
| --- | --- |
| Script injection through message text, titles, roles, names | Context-aware template escaping; raw HTML shown as escaped text; role mapped to a fixed class (REQ-HTML-4, REQ-HTML-6, REQ-SEC-1). |
| Script injection through links and images | Dangerous URL schemes dropped (REQ-HTML-5). |
| Injected or replaced script | The policy allows one script by hash and nothing else (REQ-COPY-3, REQ-SEC-2). |
| Remote content and tracking | The policy blocks all external resources; pages contain none (REQ-HTML-8). |
| Path traversal and unsafe file names | Names sanitized; `-assets` must be a single name; folder and file names unique and capped (REQ-NAME-1 to 6, REQ-SEC-3). |
| Exposure of private chat history on a shared machine | Files `0600`, directories `0700` (REQ-OUT-3). |
| Loss of existing data | No deletion; no overwrite without `-overwrite`; exclusive create (REQ-OUT-2). |
| Data leaving the machine | No network code and no telemetry (REQ-SEC-4). |
| Sensitive text in logs | Logs name attachments and keys only, never message text (REQ-SEC-5). |

Not mitigated: the content of the export itself is written verbatim into the
Markdown files, so anything that renders Markdown unsafely will show it as the
author wrote it; `-in` and `-out` are used as given; a hostile local user who
can already read the user's files is out of scope.

## 8. Error handling and logging

- Errors are values wrapped with `%w` and returned to `main`, which logs once and
  exits 1. Messages:

Every message below is a prefix: it is followed by a space and the wrapped
cause, except where noted.

| Failure | Message starts with |
| --- | --- |
| Missing `-in` | `missing required -in flag` (no cause) |
| Bad `-assets` | `invalid -assets value: must be a single directory name:` |
| Bad `-format` | `invalid -format value: must be md, html, or both:` |
| Read | `read input:` |
| Parse | `parse JSON:` |
| Empty export | `no history entries found` (no cause) |
| Output directory | `create output directory:` |
| In a conversation | `conversation <i>:` then one of `create conversation folder`, `create assets dir`, `write markdown:`, `write html:`, `render markdown:`, `render conversation page:`, or `link` |
| Index pages | one of `write index:`, `write folder index:`, `render index page:`, `render folder page:`, or `link` |
| Template parse | `parse page template:`, `parse folder template:`, or `parse index template:` |

- The error line uses the standard library's `log/slog` default text handler on
  standard error: `<date> <time> ERROR conversion failed error="<message>"`.
- A skipped attachment logs a line and continues. The date and time are in the
  form `2026/10/16 13:10:36` (local time). Fields appear in this order, and a
  value is double-quoted as by Go's `strconv.Quote` if it is empty or contains a
  space, `=`, a double quote, or a non-printable character (so every `error=`
  value, which contains spaces, is quoted):
  - Missing blob:
    `<date> <time> WARN attachment skipped name=<name> reason="missing content" fileKey=<key>`.
  - Any other failure:
    `<date> <time> WARN attachment skipped name=<name> reason="<reason>" error=<cause>`,
    where `<reason>` is `failed to load attachment payload`,
    `failed to resolve attachment path`, or `failed to write attachment`.
- A flag error is printed by the `flag` package (message and usage) and is then
  also logged as `conversion failed`, exiting with status 1.
- The two `link` messages are `link "<path>" from the index:` and
  `link "<dir>" to the main index:`. They come from a failed relative-path
  computation and cannot occur for paths under `-out`. A full message reads, for
  example, `conversation 2: write markdown: write "out/Folder/Name.md": <cause>`.
- Per-attachment failures never fail the run (REQ-ATT-4).
- There are no retries and no partial-result summary: the first fatal error ends
  the run, and files already written stay.

## 9. Technology and dependencies

- Language: Go. `go.mod` sets `go 1.26` and `toolchain go1.27.2` (the toolchain
  line is a security floor: it includes the `net/http` fixes for GO-2026-6603,
  GO-2026-6611, GO-2026-6613, and GO-2026-6617).
- Module path: `github.com/byronjones-elsevier/AIProJsonMD`.
- Runtime dependency: `github.com/yuin/goldmark v1.8.6`, used only to render
  message Markdown to HTML, with the Table, Strikethrough, and TaskList
  extensions and the custom raw-HTML renderers. It was chosen over a hand-written
  renderer and over escaped plain text (decision D-10).
- Standard library packages used include `flag`, `log/slog`, `encoding/json`,
  `encoding/base64`, `crypto/sha256`, `html/template`, `net/http` (only for
  content-type detection of image bytes), `net/url`, `path`, `path/filepath`,
  `os`, `io/fs`, `regexp`, `sort`, `unicode`, and `unicode/utf8`.
- Tools: `golangci-lint` v2 (CI pins v2.15.0; a local build must be at least as
  new as the Go toolchain); `govulncheck` v1.9.0 via `make vuln`;
  `markdownlint-cli2` 0.23.3; GNU Make.

`go.mod` is exactly (and `go.sum` holds the checksums for goldmark):

```text
module github.com/byronjones-elsevier/AIProJsonMD

go 1.26

toolchain go1.27.2

require github.com/yuin/goldmark v1.8.6
```

## 10. Build, run, test, and release

- Build: `make build`. Run: `make run IN=export.json FORMAT=html`. Full local
  check: `make all`. Vulnerability scan: `make vuln` (needs network).
- Tests use only the standard library `testing` package. They are table-driven
  where a function has many cases and use `t.TempDir()` for files.
- Lint configuration (`.golangci.yml`, version 2): the default standard
  linters plus `errorlint`, `gosec`, `misspell`, `revive`, and `unconvert`; the
  `gosec` rule G304 is excluded because the program reads and writes paths
  derived from its arguments and sanitized names.
- Markdown lint: `markdownlint-cli2` with `.markdownlint.json`
  (`MD013` off, `MD025` with an empty `front_matter_title`) over `**/*.md`,
  excluding `.git`, `out_markdown`, and `in_json`.
- CI (`.github/workflows/ci.yml`) runs on pull requests and pushes that target
  `main`, with `permissions: contents: read`, a concurrency group per
  workflow and ref that cancels superseded runs, and a 5 to 10 minute timeout per
  job. Actions are pinned to full commit SHAs and updated by Dependabot. Jobs:
  1. Test on `ubuntu-latest` and `macos-latest`: `make fmt/check vet test/race build`.
  2. Lint: `golangci-lint` v2.15.0.
  3. Vulnerability scan: `make vuln`.
  4. Markdown lint: `markdownlint-cli2@0.23.3`.
- Dependabot (`.github/dependabot.yml`) checks `github-actions` and `gomod`
  weekly.
- Release: the project has no versioned releases or published artifacts. A
  change is delivered by a pull request to `main` whose CI jobs pass.

## 11. Verification plan

### 11.1 Acceptance tests

Each requirement group is verified by these tests (all in package `main`).

| Requirements | Tests |
| --- | --- |
| REQ-CLI-1 to 6, REQ-ERR-1, 2 | `TestRunErrors`, `TestRunRejectsInvalidFormat`, `TestParseFormat`, `TestValidateAssetsDir`, `TestOutputFormatWrites`, `TestRunConvertsExport` (summary text) |
| REQ-IN-1 to 3 | `TestRunConvertsExport`, `TestStringValue`, `TestRunErrors` |
| REQ-OUT-1, 2 | `TestRunConvertsExport`, `TestRunOverwriteBehavior`, `TestRunHTMLOverwriteBehavior`, `TestRunHTMLFolderPagesFollowOverwriteRules`, `TestWriteFile_NoOverwritePreservesExisting`, `TestWriteFile_OverwriteReplacesExisting`, `TestWriteFileReportsWhetherItWrote`, `TestWriteFileMissingDirectoryFails` |
| REQ-OUT-3 | `TestRunCreatesPrivateOutput`, `TestRunHTMLOutputIsPrivate` |
| REQ-NAME-1 to 3 | `TestSanitizeFilename`, `TestTruncateName`, `TestSanitizeFilenameCapsLength`, `TestReservedWindowsNamesAreEscapedInPaths` |
| REQ-NAME-4 to 6 | `TestRunUnfiledNameIsReserved`, `TestConversationMarkdownPath_*`, `TestConversationMarkdownPathReservesIndex`, `TestBuildFolderPaths`, `TestBuildFolderPathsAvoidsCollisions`, `TestRunReservesIndexNames` |
| REQ-MD-1, 2 | `TestRunConvertsExport`, `TestRunMarkdownOnlyWritesNoHTML`, `TestRunBothFormats` |
| REQ-ATT-1 to 4 | `TestCollectConversationFiles`, `TestCollectConversationFilesEmpty`, `TestAttachmentPayload`, `TestAttachmentPayload_UsesImageBase64AndAddsExtension`, `TestDecodeBase64Payload`, `TestDecodeBase64Payload_RawBase64`, `TestMimeExtension`, `TestResolveAssetPath`, `TestMarkdownLink`, `TestRunConvertsExport` |
| REQ-HTML-1 to 9 | `TestMarkdownToHTMLRendersFormatting`, `TestMarkdownToHTMLEscapesRawHTML`, `TestMarkdownToHTMLNeutralizesDangerousURLs`, `TestRoleClass`, `TestIsImageName`, `TestRenderConversation`, `TestRenderConversationOmitsEmptySections`, `TestRunHTMLFormat`, `TestRunBothFormats` |
| REQ-NAV-1 to 4 | `TestConversationNav`, `TestFolderNav`, `TestRenderConversationShowsBreadcrumbAndLinks`, `TestRenderFolder`, `TestRenderIndexGroupsAndSorts`, `TestRunHTMLNavigation`, `TestMarkdownOnlyRunWritesNoFolderPages` |
| REQ-COPY-1 to 4, REQ-SEC-2 | `TestContentSecurityPolicy`, `TestPageWithCodeBlockGetsCopyScriptAllowedByHash`, `TestPagesWithoutCodeBlocksHaveNoScript`, `TestChatContentCannotAddScripts`, `TestCopyScriptHasNoPitfallsForTheTemplate`, `TestRunWritesCopyScriptOnlyOnPagesWithCode` |
| REQ-SEC-1, 3 | the escaping, URL, and name tests above |
| REQ-NFR-2 | `TestOnlyGoldmarkIsRequired` |
| REQ-OUT-2 (no deletion) | `TestProgramNeverDeletesFiles` |
| REQ-SEC-3 | `TestRunKeepsOutputInsideOutDir`, `TestValidateAssetsDir` |
| REQ-SEC-4 | `TestProgramMakesNoNetworkCalls` |
| REQ-NAV-3 | `TestGeneratedLinksAreRelative` |
| REQ-CLI-6, REQ-ATT-4 (warnings) | `TestAttachmentWarningIsLogged` |
| REQ-MD-1 (edge cases) | `TestRenderMarkdownEdgeCases` |
| REQ-HTML-3 to 5 (examples) | `TestSpecRenderingExamples` |
| REQ-OUT-4 | `TestRunIsDeterministic` |
| REQ-CLI-5 | `TestRunHelpReturnsErrHelp` |
| Spec drift and the worked example | `TestSpecMatchesCode`, `TestSpecStatesToolchainVersions`, `TestSpecWindowsReservedNamesMatchCode`, `TestSpecVerificationTableNamesRealTests`, `TestSpecWorkedExample` |

Requirements verified by named procedures instead of an automated test:

- REQ-HTML-8 and REQ-HTML-9 (self-contained pages, light and dark) are verified
  by the page audit in section 11.2 and by viewing pages in a browser.
- REQ-COPY-4 (behavior of the script): the tests check the script text,
  placement, and policy statically; nothing runs the script in an automated
  test, so its behavior was verified by hand in a browser (see CHANGELOG.md).
- REQ-CLI-5 and the exit statuses in 3.3: `TestRunHelpReturnsErrHelp` checks the
  help path returns the error that `main` treats as success, and `main` is three
  statements that were reviewed. No test runs the built binary.
- REQ-SEC-5 (no message text in logs): verified by review; every log call is in
  `conversation.go` and `main.go` and names only attachment names and keys.

- REQ-NFR-1, 3, 4 are verified by CI (section 10) and by `make all`.
- REQ-NFR-2 (only goldmark) is verified by `TestOnlyGoldmarkIsRequired`.

All other requirements have an automated test in the table above.

`TestSpecEveryRequirementIsVerified` fails if a requirement in section 2 is not
mentioned in this section, which is how REQ-NFR-5 is checked.

### 11.2 Validation against realistic data

Before merging a change that affects output, run the previous build and the new
build on a representative export (at least dozens of conversations with
folders, attachments, code blocks, tables, and raw HTML) and compare:

1. Build a baseline binary from `main` and a binary from the branch.
2. Run both with the same flags into separate empty directories.
3. For Markdown, `diff -r` the trees excluding `*.html`. Any difference MUST be
   explained by the change.
4. For HTML, audit every page: no `script`, `iframe`, `object`, `embed`, `form`,
   `base`, or `link` elements except the one allowed script; no event-handler
   attributes; every page has the policy; every script's hash matches its policy;
   every local link and image resolves; each non-index page has a breadcrumb with
   one current-page crumb.
5. Run the new binary twice into separate directories and `diff -r` them
   (determinism).

Do not print or keep private content from the export; report counts and
pass or fail.

### 11.3 Required checks and coverage

`make all` MUST pass with 0 lint issues, and CI MUST pass on the pull request.
Statement coverage (`make test/cover`) SHOULD stay at or above 85%.

### 11.4 Keeping this specification true

`TestSpecMatchesCode` reads this file and fails if: the stylesheet, head,
navigation block, page tails, or script in the appendices differ from the
constants in the code; the Content-Security-Policy base string differs; the
`go`, `toolchain`, and goldmark versions in `go.mod` do not appear in section 9;
or a Windows reserved name in the code is missing from section 5.3. Run the
test after any change to those, and update this file in the same change.

## 12. Decisions

Each decision records the context, the choice, the reasons, rejected
alternatives, and consequences. A superseded decision stays, marked as such.

- D-01 Conversation file names come from the conversation name (accepted).
  Context: the first version named files by hash plus index. Choice: use the
  sanitized name, falling back to the hash form for unnamed conversations.
  Rejected: hash names, which are stable but unreadable. Consequence: names can
  collide, so section 5.4 adds suffixes; names can change if titles change, so a
  re-run into an old directory without `-overwrite` can leave old and new names
  side by side.
- D-02 One flat assets directory per folder with collision handling (accepted).
  Identical content is reused; different content gets `_2`, `_3`. Rejected: a
  directory per conversation, which lengthens links and deepens the tree.
- D-03 No overwrite by default, exclusive create (accepted). It avoids a
  check-then-write race and never destroys existing work. Consequence: refreshing
  output requires `-overwrite`, including for index pages.
- D-04 Fail fast on fatal errors, degrade on attachments (accepted). Rejected:
  collecting every error into a summary, which adds complexity to a one-shot tool.
- D-05 Standard library `testing` only (accepted). The suite is table-driven and
  readable without assertion helpers, and the dependency set stays what the
  product needs. This overrides the global `testify` preference on purpose.
  Rejected: `testify`, which would add a test-only dependency and about 60
  rewritten assertions for little gain.
- D-06 `-assets` is a single directory name (accepted). Rejected: nested relative
  paths, which need containment checks and complicate links with no known use.
- D-07 Private output, `0600` and `0700` (accepted), because chat history can be
  sensitive. Applies only to files and directories the program creates.
- D-08 Names are capped at 200 bytes (accepted), leaving room for `.md`, `.html`,
  and `_N` suffixes under the common 255-byte filename limit.
- D-09 Windows-safe names (accepted): reserved device names are prefixed with
  `_`, and trailing spaces and dots are trimmed, because Windows rejects them.
  Consequence: titles that end in a space (for example truncated chat titles)
  produce a different file name than earlier versions; the content is identical.
  CI does not run on Windows.
- D-10 HTML export uses goldmark (accepted). Rejected: a hand-written renderer,
  which would stay dependency-free but must reproduce fenced code, nested lists,
  and tables correctly; and escaped plain text, which is safest but hard to read.
  Consequence: the project's first third-party dependency, so `go.sum`,
  Dependabot `gomod` updates, and `govulncheck` cover it.
- D-11 `-format md|html|both`, default `md` (accepted), so existing invocations
  are unchanged. Rejected: a boolean `-html` flag, which cannot produce HTML
  only.
- D-12 Three-level HTML hierarchy with a breadcrumb as the only navigation
  (accepted). The folder page gives "up one level" a real target. A second line of
  "Up to folder" and "Main index" links was tried and removed because it repeated
  the breadcrumb. All links are relative so the tree works from disk.
- D-13 `index` is a reserved conversation name (accepted), because each folder's
  page is `index.html`. The reservation applies to every format so Markdown and
  HTML names match; a Markdown-only run therefore also renames a conversation
  titled `index`.
- D-14 Folder directory names are unique ignoring case, and `index.html` and
  `Unfiled` are reserved (accepted), so `Docs` and `docs` do not share a
  directory on case-insensitive filesystems, a folder cannot collide with the
  main index, and `Unfiled` holds only unmapped conversations. Earlier, an export
  folder named `Unfiled` was merged into that directory; the owner chose to give
  such a folder a numeric suffix (`Unfiled_2`) instead, as for any other conflict.
  Consequence: an export folder named `Unfiled` appears as `Unfiled_2`.
- D-15 Single newlines render as line breaks (accepted). Plain Markdown joins
  lines with a space, but chat messages use newlines as typed. Cost: a message
  hard-wrapped at a fixed column in its source would show ragged line ends.
- D-16 Raw HTML in messages is shown as escaped text (accepted). goldmark's safe
  mode alone would replace it with a comment and lose the text; unsafe mode
  would allow injection.
- D-17 Copy buttons are a hash-pinned inline script (accepted). The first
  JavaScript in the output. Included only on pages with a code block; the policy
  allows exactly that script by hash. Buttons are added by script so a page
  without script has no dead buttons. Rejected: `script-src 'unsafe-inline'`
  (any injected script would run), an external `.js` file (adds a file to every
  output tree), server-rendered buttons (dead without script), and the script on
  every page (grants script where it is not needed). The fallback to
  `execCommand` covers `file://` pages in browsers that restrict the Clipboard
  API.
- D-18 One extraction, two renderers (accepted). Attachments are extracted once
  into a neutral list that both renderers read, so Markdown stays identical
  whether or not HTML is written and the formats cannot disagree about names.
- D-19 No `context.Context` parameters (accepted): the program does short local
  file work with nothing cancellable to propagate.
- D-20 Pinned `govulncheck` through `go run ...@version` in the Makefile
  (accepted), so nothing is installed globally and the version is reproducible.
- D-21 The `toolchain` line in `go.mod` is a security floor (accepted): `go`
  fetches a patched toolchain when the local one is older.
- D-22 CI runs on Linux and macOS, not Windows (accepted for now): the tests have
  not been confirmed on Windows. See HANDOFF.md.
- D-23 `ENGINEERING.md` is the primary specification, with `HANDOFF.md` for
  outstanding work and `CHANGELOG.md` for history (accepted), following the
  global standard in `agents/coding.md`. A test keeps the specification's
  appendices in step with the code.

## Appendix A: stylesheet

The stylesheet is inserted verbatim between `<style>` and `</style>` on every
page.

```css
:root{color-scheme:light dark;--bg:#fff;--fg:#1f2328;--muted:#59636e;--border:#d1d9e0;--user:#f6f8fa;--code:#f6f8fa}
@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--fg:#e6edf3;--muted:#9198a1;--border:#3d444d;--user:#161b22;--code:#161b22}}
body{margin:0;background:var(--bg);color:var(--fg);font:16px/1.6 system-ui,-apple-system,"Segoe UI",sans-serif}
main{max-width:52rem;margin:0 auto;padding:2rem 1rem}
.meta{color:var(--muted)}
.message{border:1px solid var(--border);border-radius:8px;padding:.5rem 1rem;margin:1rem 0}
.message h3{margin:.25rem 0;font-size:.85rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted)}
.role-user{background:var(--user)}
.body{overflow-wrap:anywhere}
pre{background:var(--code);border:1px solid var(--border);border-radius:6px;padding:.75rem;overflow:auto}
code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:.9em}
:not(pre)>code{background:var(--code);padding:.1em .3em;border-radius:4px}
table{border-collapse:collapse;display:block;overflow:auto}
th,td{border:1px solid var(--border);padding:.3rem .6rem}
blockquote{margin:0;padding-left:1rem;border-left:4px solid var(--border);color:var(--muted)}
img{max-width:100%}
.breadcrumb ol{display:flex;flex-wrap:wrap;gap:.25rem;margin:0 0 1rem;padding:0;list-style:none;color:var(--muted)}
.breadcrumb li+li::before{content:"\203A";margin-right:.25rem}
.breadcrumb li[aria-current]{color:var(--fg);overflow-wrap:anywhere}
.code-block{position:relative}
.copy-button{position:absolute;top:.4rem;right:.4rem;padding:.15rem .5rem;font:inherit;font-size:.8rem;color:var(--fg);background:var(--bg);border:1px solid var(--border);border-radius:4px;cursor:pointer;opacity:.75}
.copy-button:hover,.copy-button:focus-visible{opacity:1}
.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0);white-space:nowrap}
@media print{.copy-button{display:none}}
```

## Appendix B: page templates

The templates are Go `html/template` text. In the head, `{{pageStyle}}` stands
for the stylesheet in Appendix A. `{{.CSP}}` is the policy from section 5.9.

The head, shared by every page:

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="{{.CSP}}">
<title>{{.Title}}</title>
<style>{{pageStyle}}</style>
</head>
<body>
<main>
```

The breadcrumb block, shared by conversation and folder pages:

```html
<nav class="breadcrumb" aria-label="Breadcrumb">
<ol>
{{range .Nav.Crumbs}}<li{{if not .Href}} aria-current="page"{{end}}>{{if .Href}}<a href="{{.Href}}">{{.Label}}</a>{{else}}{{.Label}}{{end}}</li>
{{end}}</ol>
</nav>
```

A conversation page is the head, the breadcrumb block, then:

```html
<h1>{{.Title}}</h1>
{{if .ID}}<p class="meta">id: <code>{{.ID}}</code></p>
{{end}}<h2>Messages</h2>
{{range .Messages}}<section class="message {{.RoleClass}}">
<h3>{{.Role}}</h3>
<div class="body">
{{.Body}}</div>
</section>
{{end}}{{if .Attachments}}<h2>Attachments</h2>
<ul class="attachments">
{{range .Attachments}}<li><strong>{{.Name}}</strong>: {{if .Failure}}<em>{{.Failure}}</em>{{else}}<a href="{{.Href}}">{{if .IsImage}}<img src="{{.Href}}" alt="{{.Name}}" loading="lazy"><br>{{end}}{{.LinkText}}</a>{{end}}</li>
{{end}}</ul>
{{end}}</main>
{{if .Script}}<script>{{.Script}}</script>
{{end}}</body>
</html>
```

A folder page is the head, the breadcrumb block, then:

```html
<h1>{{.Title}}</h1>
<ul>
{{range .Conversations}}<li><a href="{{.Href}}">{{.Title}}</a></li>
{{end}}</ul>
</main>
</body>
</html>
```

The main index is the head, then:

```html
<h1>{{.Title}}</h1>
{{range .Folders}}<h2><a href="{{.Href}}">{{.Name}}</a></h2>
<ul>
{{range .Conversations}}<li><a href="{{.Href}}">{{.Title}}</a></li>
{{end}}</ul>
{{end}}</main>
</body>
</html>
```

## Appendix C: copy script

The script is inserted verbatim between `<script>` and `</script>`.

```javascript
(function () {
  var blocks = document.querySelectorAll("pre > code");
  if (!blocks.length) { return; }

  function legacyCopy(text) {
    return new Promise(function (resolve, reject) {
      var area = document.createElement("textarea");
      area.value = text;
      area.setAttribute("readonly", "");
      area.style.position = "fixed";
      area.style.opacity = "0";
      document.body.appendChild(area);
      area.select();
      var ok = false;
      try { ok = document.execCommand("copy"); } catch (err) { ok = false; }
      document.body.removeChild(area);
      if (ok) { resolve(); } else { reject(new Error("copy failed")); }
    });
  }

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text).catch(function () { return legacyCopy(text); });
    }
    return legacyCopy(text);
  }

  Array.prototype.forEach.call(blocks, function (code) {
    var pre = code.parentNode;
    var wrap = document.createElement("div");
    wrap.className = "code-block";
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);

    var button = document.createElement("button");
    button.type = "button";
    button.className = "copy-button";
    button.textContent = "Copy";
    button.setAttribute("aria-label", "Copy code to clipboard");

    var status = document.createElement("span");
    status.className = "sr-only";
    status.setAttribute("role", "status");

    var timer;
    function report(label, message) {
      button.textContent = label;
      status.textContent = message;
      clearTimeout(timer);
      timer = setTimeout(function () {
        button.textContent = "Copy";
        status.textContent = "";
      }, 2000);
    }

    button.addEventListener("click", function () {
      copyText(code.textContent).then(
        function () { report("Copied!", "Code copied to clipboard"); },
        function () { report("Failed", "Copy failed"); }
      );
    });

    wrap.appendChild(button);
    wrap.appendChild(status);
  });
})();
```
