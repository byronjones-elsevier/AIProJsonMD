# Engineering Notes

## Scope

This repository contains a single Go CLI, `AIProJsonMD`, that converts an exported AI chat JSON file into:

- one Markdown file per conversation
- one output subfolder per export folder name, plus `Unfiled` for chats without a mapped folder
- one local flat `assets` directory inside each output folder

The implementation currently lives entirely in `main.go`.

## Build And Run

```bash
go build -o AIProJsonMD
go run . -in aipro_chat_history_2026-06-29-1248.json -out out_markdown
go run . -in aipro_chat_history_2026-06-29-1248.json -out out_markdown -overwrite
```

## Data Model

The parser expects three main shapes:

- `Export`: top-level document with `history` and `files`
- `Conversation`: one chat thread with metadata, messages, optional conversation-level file refs, and an optional `folderId`
- `Folder`: top-level folder metadata keyed by folder ID
- `FileBlob`: top-level attachment payload keyed by `fileKey`

Attachment references can appear in two places:

- `history[].files`
- `history[].messages[].files`

The converter merges both sources for each conversation, de-duplicates them, and resolves each reference against the top-level `files` collection.

Folder placement is driven by:

- `history[].folderId`
- `folders[].id` and `folders[].name`

## Current Processing Flow

1. Parse the input JSON into `Export`.
2. Build a `map[string]FileBlob` by `fileKey` from the top-level `files` array.
3. Build a folder ID to output path map from the top-level `folders` array.
4. For each conversation:
5. Resolve the output folder from `folderId`, or fall back to `out_markdown/Unfiled/`.
6. Compute a short hash from the conversation ID.
7. Merge conversation-level and message-level attachment refs.
8. Write one Markdown file containing title, ID, messages, and attachment links.
9. Write attachment payloads under `<folder>/assets/`.

## Schema Quirks Observed

- `fileSize` is not stable across the export and may be either a string or a number.
- Some conversations have empty `files` arrays even when message-level attachments exist.
- Some conversations have `folderId: null`, so they fall back to `Unfiled`.
- Attachment payloads may be in `files[].content` or `files[].imageBase64`.
- `fileType` can help infer an image file extension when the attachment name does not include one.

## Current Limitations

- Base64 image payloads are decoded and written as binary files when `content` is empty.
- Existing markdown and attachment files are intentionally not overwritten unless `-overwrite` is enabled.
- Flattened per-folder assets can create filename collisions, so the program appends numeric suffixes when needed.
- Image extension inference currently handles common web image types via `fileType` or `http.DetectContentType`.
- Markdown filenames use a hash plus ordinal index rather than the conversation title.
- Duplicate export folder names are disambiguated with a numeric suffix in the output tree.
- The program exits on parse or write failures instead of collecting partial error summaries.

## Verification

Validated in this workspace with:

```bash
go build
go run . -in aipro_chat_history_2026-06-29-1248.json -out out_markdown
```

Observed generated folders include:

- `out_markdown/Bash Projects/`
- `out_markdown/General Research/`
- `out_markdown/GoLang Projects/`
- `out_markdown/Jenkins Diagnostics/`
- `out_markdown/Jenkins Projects/`
- `out_markdown/Javascript Bookmarklets/`
- `out_markdown/Unfiled/`

Observed extracted files include:

- `out_markdown/Unfiled/assets/consoleText.txt`
- `out_markdown/Unfiled/assets/tls_inventory.json`
- `out_markdown/Unfiled/assets/aipro_chat_history_2026-06-29-1232.json`
- `out_markdown/Unfiled/assets/TIO_All_Hands_20260610.txt`

Those extracted files are currently under `Unfiled` because the source conversations in the sample export have `folderId: null`.

## About AGENTS.md

An `AGENTS.md` file is not currently necessary for this repository. The project is small, has one executable entry point, and the build and runtime expectations are covered by `README.md` and this document.