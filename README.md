# AIProJsonMD

`AIProJsonMD` is a small Go command-line tool that converts an AI chat export JSON file into a set of Markdown files, grouped by export folder, with attachments written into a local `assets` directory inside each output folder.

## What It Does

- Reads a JSON export file with conversation history and file blobs.
- Writes one Markdown file per conversation.
- Groups conversations into subfolders under `out_markdown/` using the export `folderId` to top-level `folders` mapping.
- Preserves message roles and content.
- Places conversations without a mapped folder into `out_markdown/Unfiled/`.
- Extracts attachment content into a local flat `assets/` directory inside each output folder.
- Collects attachment references from both conversation-level `files` arrays and message-level `files` arrays.
- Decodes `imageBase64` payloads into image files when an attachment blob does not have plain-text `content`.
- Adds relative links from each Markdown file to its extracted attachments.

## Requirements

- Go 1.26 or newer
- Optional, for `make lint` and `make vuln`: `golangci-lint` and `govulncheck`

## Build

```bash
make build
```

This produces a local executable named `AIProJsonMD` in the project root.
Without `make`, run `go build -o AIProJsonMD .`.

## Run

```bash
make run IN=export.json
```

Or run the built binary directly:

```bash
./AIProJsonMD -in export.json -out out_markdown
```

Progress and the final summary go to stdout. Skipped attachments and fatal
errors are logged to stderr, and a fatal error exits with status 1.

## Flags

- `-in`: path to the input JSON export file. Required.
- `-out`: directory for generated Markdown files. Default: `out_markdown`
- `-assets`: name of the per-folder assets directory created inside each output folder. Default: `assets`. Must be a single directory name: empty values, `.`, `..`, and anything containing a path separator are rejected.
- `-overwrite`: overwrite existing markdown and attachment files. Default: `false`

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

- Markdown files are written to named subfolders under `out_markdown/`
- Attachments are written to `assets/` inside the same folder as the related markdown files

Example:

```text
out_markdown/
  Bash Projects/
    Deploy script review.md
    Deploy script review_2.md
    assets/
      notes.txt
  Unfiled/
    2b3c4d5e6f_2.md
```

## Notes

- Conversation file names come from the sanitized conversation name. A duplicate name in the same folder gets a `_2`, `_3`, ... suffix. A conversation with no name falls back to a short SHA-256 hash of its ID plus its position in the export.
- Folder names are taken from the export `folders` list and sanitized for filesystem use.
- Attachment references are de-duplicated per conversation before extraction.
- Attachment file names are sanitized for filesystem safety.
- If multiple attachments in the same folder would produce the same filename, the program appends a numeric suffix to keep them distinct.
- If an image attachment has no file extension, the program attempts to infer one from `fileType` or the decoded bytes.
- Existing markdown and attachment files are not overwritten unless `-overwrite` is set.
- If an attachment reference exists in a conversation but no matching file blob is found, the Markdown output notes the missing attachment.

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

Tests use the standard library `testing` package and live next to the code.
`run` is covered end to end with a synthetic export built in the test, so no
real chat history is needed.

## Engineering Notes

See [ENGINEERING.md](ENGINEERING.md) for architecture, key decisions,
verification status, and known risks. Agent guidance lives in
[Agents.md](Agents.md).
