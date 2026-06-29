# AIChatJsonMD

`AIChatJsonMD` is a small Go command-line tool that converts an AI chat export JSON file into a set of Markdown files, grouped by export folder, with attachments written into a local `assets` directory inside each output folder.

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

- Go 1.22 or newer

## Build

```bash
go build
```

This produces a local executable named `aichatjsonmd` in the project root.

## Run

```bash
go run . -in aipro_chat_history_2026-06-29-1248.json -out out_markdown
```

Or run the built binary:

```bash
./aichatjsonmd -in aipro_chat_history_2026-06-29-1248.json -out out_markdown
```

## Flags

- `-in`: path to the input JSON export file. Required.
- `-out`: directory for generated Markdown files. Default: `out_markdown`
- `-assets`: name of the per-folder assets directory created inside each output folder. Default: `assets`
- `-overwrite`: overwrite existing markdown and attachment files. Default: `false`

## Expected Input Shape

The program expects a JSON document with this general structure:

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
    1a2b3c4d5e_1.md
    assets/
      notes.txt
  Unfiled/
    2b3c4d5e6f_2.md
```

## Notes

- Conversation file names use a short SHA-256 hash derived from the conversation ID plus the conversation index.
- Folder names are taken from the export `folders` list and sanitized for filesystem use.
- Attachment references are de-duplicated per conversation before extraction.
- Attachment file names are sanitized for filesystem safety.
- If multiple attachments in the same folder would produce the same filename, the program appends a numeric suffix to keep them distinct.
- If an image attachment has no file extension, the program attempts to infer one from `fileType` or the decoded bytes.
- Existing markdown and attachment files are not overwritten unless `-overwrite` is set.
- If an attachment reference exists in a conversation but no matching file blob is found, the Markdown output notes the missing attachment.

## Engineering Notes

See [ENGINEERING.md](ENGINEERING.md) for implementation details, schema assumptions, and maintenance notes.