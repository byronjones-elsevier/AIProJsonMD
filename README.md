# AIProJsonMD

`AIProJsonMD` is a small Go command-line tool that converts an AI chat export JSON file into a set of Markdown files, HTML files, or both, grouped by export folder, with attachments written into a local `assets` directory inside each output folder.

## What It Does

- Reads a JSON export file with conversation history and file blobs.
- Writes one Markdown file per conversation (default), one HTML file per conversation, or both, selected with `-format`.
- With HTML output, also writes an `index.html` that links every conversation, grouped by folder.
- Groups conversations into subfolders under `out_markdown/` using the export `folderId` to top-level `folders` mapping.
- Preserves message roles and content.
- Places conversations without a mapped folder into `out_markdown/Unfiled/`.
- Extracts attachment content into a local flat `assets/` directory inside each output folder.
- Collects attachment references from both conversation-level `files` arrays and message-level `files` arrays.
- Decodes `imageBase64` payloads into image files when an attachment blob does not have plain-text `content`.
- Adds relative links from each Markdown or HTML file to its extracted attachments, and shows image attachments inline in HTML.

## Requirements

- Go 1.26 or newer
- One Go module dependency, `github.com/yuin/goldmark`, fetched automatically by `go build`
- Optional, for `make lint` and `make vuln`: `golangci-lint` and `govulncheck`

## Build

```bash
make build
```

This produces a local executable named `AIProJsonMD` in the project root.
Without `make`, run `go build -o AIProJsonMD .`.

## Run

```bash
make run IN=export.json                  # Markdown (default)
make run IN=export.json FORMAT=html      # HTML and index.html
make run IN=export.json FORMAT=both      # Markdown and HTML
```

Or run the built binary directly:

```bash
./AIProJsonMD -in export.json -out out_markdown
```

Progress and the final summary go to stdout. Skipped attachments and fatal
errors are logged to stderr, and a fatal error exits with status 1.

## Flags

- `-in`: path to the input JSON export file. Required.
- `-out`: directory for generated files. Default: `out_markdown`
- `-format`: output format. `md` writes Markdown only, `html` writes HTML and `index.html` only, `both` writes all of them. Default: `md`
- `-assets`: name of the per-folder assets directory created inside each output folder. Default: `assets`. Must be a single directory name: empty values, `.`, `..`, and anything containing a path separator are rejected.
- `-overwrite`: overwrite existing output and attachment files, including `index.html`. Default: `false`

## Expected Input Shape

The program expects an "AI Pro Export" JSON document with this general structure:

```json
{
  "version": 1,
  "history": [
    {
      "id": "conversation-id",
      "name": "Conversation title",
      "folderId": "folder-1",
      "messages": [
        {
          "role": "user",
          "content": "Hello",
          "files": [
            {
              "fileKey": "file-1",
              "fileName": "notes.txt",
              "fileSize": 7229
            }
          ]
        }
      ],
      "files": [
        {
          "fileKey": "file-1",
          "fileName": "notes.txt",
          "fileSize": "7.06 KB"
        }
      ]
    }
  ],
  "folders": [
    {
      "id": "folder-1",
      "name": "Project A",
      "type": "chat"
    }
  ],
  "files": [
    {
      "fileKey": "file-1",
      "fileName": "notes.txt",
      "fileSize": "7.06 KB",
      "conversationId": "conversation-id",
      "content": "attachment content"
    }
  ]
}
```

Notes about the export schema:

- Conversation attachments may be listed on the conversation object, on individual messages, or both.
- Conversations may reference a folder by `folderId`, and the export defines folder names in the top-level `folders` array.
- `fileSize` may be a human-readable string like `"7.06 KB"` or a numeric value.
- Attachment payloads are written from `files[].content` when present, or from decoded `files[].imageBase64` when `content` is empty.

## Output Layout

After a successful run:

- Markdown and HTML files are written to named subfolders under `out_markdown/`
- Attachments are written to `assets/` inside the same folder as the related conversation files
- With `-format html` or `-format both`, `index.html` is written at the top of the output directory

Example with `-format both`:

```text
out_markdown/
  index.html                      main index: every folder and conversation
  Bash Projects/
    index.html                    folder page: this folder's conversations
    Deploy script review.md
    Deploy script review.html
    Deploy script review_2.md
    Deploy script review_2.html
    assets/
      notes.txt
  Unfiled/
    index.html
    2b3c4d5e6f_2.md
    2b3c4d5e6f_2.html
```

## HTML Output

Each HTML file is a standalone page with embedded CSS, no JavaScript, and a light or dark theme that follows your system setting. Message text is rendered from Markdown, including fenced code blocks, lists, tables, headings, links, and strikethrough. Image attachments appear inline; other attachments are links into `assets/`.

Chat content is treated as untrusted:

- Raw HTML written in a message, such as `<script>` or `<div>`, is shown as literal escaped text. It is never executed or dropped.
- Links and images using `javascript:`, `vbscript:`, `file:`, or non-image `data:` URLs are rendered without a link.
- Each page includes a Content-Security-Policy that blocks scripts and all external resources, so remote images in a chat are not loaded.

### Navigation

The pages form three levels: the main index, a page for each folder, and the conversations. Every page below the main index shows a breadcrumb of where you are, with the current page last and not linked:

```text
Main index › Bash Projects › Deploy script review
```

- The breadcrumb is the navigation; there is no separate row of links. Every crumb except the current page is a link, so the crumb just before the current page goes up one level, and the first crumb always goes back to the main index.
- A folder page lists that folder's conversations.
- The main index lists every folder, each linking to its folder page, with the conversations under it, alphabetically with `Unfiled` last.

All links are relative, so the output directory can be moved, zipped, or opened straight from disk. Like every other output file, the main index and the folder pages are kept if they already exist; use `-overwrite` to regenerate them after adding conversations.

## Notes

- Conversation file names come from the sanitized conversation name. A duplicate name in the same folder gets a `_2`, `_3`, ... suffix. A conversation with no name falls back to a short SHA-256 hash of its ID plus its position in the export.
- Folder names are taken from the export `folders` list and sanitized for filesystem use.
- Attachment references are de-duplicated per conversation before extraction.
- Attachment file names are sanitized for filesystem safety. Windows reserved device names such as `CON` or `NUL` get a leading underscore, so output can be copied to Windows.
- If multiple attachments in the same folder would produce the same filename, the program appends a numeric suffix to keep them distinct.
- If an image attachment has no file extension, the program attempts to infer one from `fileType` or the decoded bytes.
- Existing markdown and attachment files are not overwritten unless `-overwrite` is set.
- If an attachment reference exists in a conversation but no matching file blob is found, the Markdown output notes the missing attachment.

- The name `index` is reserved in every folder, because each folder has an `index.html` page. A conversation titled `index` is written as `index_2`, in Markdown too, so the two formats always use the same names.
- Folder names are compared without regard to case, so `Docs` and `docs` become `Docs` and `docs_2` instead of sharing one directory on macOS or Windows. A folder named `index.html` is renamed so it cannot collide with the main index file.
- Attachment links are URL-escaped, so names with spaces or parentheses still resolve.
- Output is private to your user: new files are created with mode `0600` and new directories with `0700`, because chat history can be sensitive. Existing files and directories keep their current permissions.
- File and folder names are capped at 200 bytes, keeping the extension, so very long conversation names do not fail the write.
- The summary line reports how many Markdown files were actually written. Files skipped because they already exist are not counted.

## Development

```bash
make all         # format check, vet, golangci-lint, race tests, build
make test        # unit tests
make test/cover  # tests with a coverage total
make vuln        # pinned govulncheck (set GOVULNCHECK_VERSION to change)
make help        # list every target
```

GitHub Actions (`.github/workflows/ci.yml`) runs the format check, vet, race
tests, and build on Linux and macOS, plus `golangci-lint`, `govulncheck`, and
Markdown lint, on every pull request and push to `main`.

Tests use the standard library `testing` package and live next to the code.
`run` is covered end to end with a synthetic export built in the test, so no
real chat history is needed.

## Engineering Notes

See [ENGINEERING.md](ENGINEERING.md) for architecture, key decisions,
verification status, and known risks. Agent guidance lives in
[Agents.md](Agents.md).
