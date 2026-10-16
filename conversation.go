package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// attachment is an extracted attachment, or the reason it could not be extracted.
// Rel is the slash-separated path from the conversation's folder to the file.
type attachment struct {
	Name    string
	Rel     string
	Failure string
}

// conversionResult reports which files convertConversation wrote. A file that
// already existed and was kept is not counted as written.
type conversionResult struct {
	markdown bool
	html     bool
}

// convertConversation extracts the conversation's attachments once, then writes
// its Markdown and/or HTML file according to the converter's format.
// Per-attachment failures are recorded in the output and logged, not returned.
func (cv *converter) convertConversation(index int, c Conversation) (conversionResult, error) {
	var res conversionResult

	folderDir := defaultConversationFolder(cv.outDir)
	if folderPath, ok := cv.folderPaths[stringValue(c.FolderID)]; ok {
		folderDir = folderPath
	}
	if err := os.MkdirAll(folderDir, dirPerm); err != nil {
		return res, fmt.Errorf("create conversation folder %q: %w", folderDir, err)
	}

	title := strings.TrimSpace(c.Name)
	if title == "" {
		title = fmt.Sprintf("Conversation %d", index+1)
	}
	id := strings.TrimSpace(c.ID)
	mdPath := conversationMarkdownPath(folderDir, c, index, cv.usedNames)

	attachments, err := cv.extractAttachments(filepath.Join(folderDir, cv.assetsDir), collectConversationFiles(c))
	if err != nil {
		return res, err
	}

	if cv.format.writesMarkdown() {
		res.markdown, err = writeFile(mdPath, []byte(renderMarkdown(title, id, c.Messages, attachments)), cv.overwrite)
		if err != nil {
			return res, fmt.Errorf("write markdown: %w", err)
		}
	}
	if cv.format.writesHTML() {
		htmlPath := strings.TrimSuffix(mdPath, ".md") + ".html"
		body, err := cv.html.renderConversation(title, id, c.Messages, attachments)
		if err != nil {
			return res, err
		}
		res.html, err = writeFile(htmlPath, body, cv.overwrite)
		if err != nil {
			return res, fmt.Errorf("write html: %w", err)
		}

		rel, err := filepath.Rel(cv.outDir, htmlPath)
		if err != nil {
			return res, fmt.Errorf("link %q from the index: %w", htmlPath, err)
		}
		cv.index = append(cv.index, indexEntry{
			Folder: filepath.Base(folderDir),
			Title:  title,
			Href:   escapeLinkPath(filepath.ToSlash(rel)),
		})
	}

	return res, nil
}

// writeIndex writes index.html linking every HTML conversation written by this
// run. It reports whether the file was written; an existing index is kept
// unless overwrite is set. It does nothing unless the format includes HTML.
func (cv *converter) writeIndex() (bool, error) {
	if !cv.format.writesHTML() {
		return false, nil
	}

	body, err := cv.html.renderIndex(cv.index)
	if err != nil {
		return false, err
	}
	written, err := writeFile(filepath.Join(cv.outDir, "index.html"), body, cv.overwrite)
	if err != nil {
		return false, fmt.Errorf("write index: %w", err)
	}
	return written, nil
}

// renderMarkdown renders one conversation as a Markdown document.
func renderMarkdown(title, id string, messages []Message, attachments []attachment) string {
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	if id != "" {
		fmt.Fprintf(&b, "_id: `%s`_\n\n", id)
	}

	b.WriteString("## Messages\n\n")
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role == "" {
			role = "message"
		}
		b.WriteString("### " + role + "\n\n")
		b.WriteString(strings.TrimSpace(m.Content) + "\n\n")
	}

	if len(attachments) > 0 {
		b.WriteString("## Attachments\n\n")
		for _, a := range attachments {
			if a.Failure != "" {
				fmt.Fprintf(&b, "- **%s**: _%s_\n", a.Name, a.Failure)
				continue
			}
			fmt.Fprintf(&b, "- **%s**: %s\n", a.Name, markdownLink(path.Base(a.Rel), a.Rel))
		}
	}

	return b.String()
}

// extractAttachments writes each referenced attachment into assetsDir and
// returns one entry per reference, in order. It returns an error only if
// assetsDir cannot be created.
func (cv *converter) extractAttachments(assetsDir string, refs []FileRef) ([]attachment, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(assetsDir, dirPerm); err != nil {
		return nil, fmt.Errorf("create assets dir %q: %w", assetsDir, err)
	}

	attachments := make([]attachment, 0, len(refs))
	for idx, ref := range refs {
		attachments = append(attachments, cv.extractAttachment(assetsDir, idx, ref))
	}
	return attachments, nil
}

// extractAttachment writes one attachment and returns where it went, or why it
// could not be written.
func (cv *converter) extractAttachment(assetsDir string, idx int, ref FileRef) attachment {
	key := strings.TrimSpace(ref.FileKey)
	origName := strings.TrimSpace(ref.FileName)
	if origName == "" {
		origName = fmt.Sprintf("attachment_%d", idx+1)
	}

	fail := func(msg string, err error) attachment {
		slog.Warn("attachment skipped", "name", origName, "reason", msg, "error", err)
		return attachment{Name: origName, Failure: fmt.Sprintf("%s: %v", msg, err)}
	}

	blob, ok := cv.fileByKey[key]
	if !ok {
		slog.Warn("attachment skipped", "name", origName, "reason", "missing content", "fileKey", key)
		return attachment{Name: origName, Failure: fmt.Sprintf("missing content for fileKey `%s`", key)}
	}

	contentBytes, resolvedName, err := attachmentPayload(blob, origName, idx)
	if err != nil {
		return fail("failed to load attachment payload", err)
	}

	safeName := sanitizeFilename(resolvedName)
	if safeName == "" {
		safeName = fmt.Sprintf("attachment_%d", idx+1)
	}

	dst, err := resolveAssetPath(assetsDir, safeName, contentBytes)
	if err != nil {
		return fail("failed to resolve attachment path", err)
	}
	if _, err := writeFile(dst, contentBytes, cv.overwrite); err != nil {
		return fail("failed to write attachment", err)
	}

	return attachment{Name: origName, Rel: path.Join(cv.assetsDir, filepath.Base(dst))}
}
