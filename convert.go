package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// linkPathEscaper escapes the parentheses that url.PathEscape leaves intact,
// since they would end a Markdown link destination early.
var linkPathEscaper = strings.NewReplacer("(", "%28", ")", "%29")

// linkTextEscaper escapes brackets that would end Markdown link text early.
var linkTextEscaper = strings.NewReplacer("[", `\[`, "]", `\]`)

// converter turns export conversations into Markdown files and extracted attachments.
type converter struct {
	outDir      string
	assetsDir   string
	overwrite   bool
	folderPaths map[string]string
	fileByKey   map[string]FileBlob
	usedNames   map[string]int
}

// newConverter indexes the export's folders and attachment blobs for lookup during conversion.
func newConverter(exp Export, outDir, assetsDir string, overwrite bool) *converter {
	fileByKey := make(map[string]FileBlob, len(exp.Files))
	for _, fb := range exp.Files {
		if key := strings.TrimSpace(fb.FileKey); key != "" {
			fileByKey[key] = fb
		}
	}

	return &converter{
		outDir:      outDir,
		assetsDir:   assetsDir,
		overwrite:   overwrite,
		folderPaths: buildFolderPaths(outDir, exp.Folders),
		fileByKey:   fileByKey,
		usedNames:   make(map[string]int, len(exp.History)),
	}
}

// collectConversationFiles returns the unique file references from a
// conversation and its messages, in first-seen order.
func collectConversationFiles(c Conversation) []FileRef {
	refs := make([]FileRef, 0, len(c.Files))
	seen := make(map[string]struct{})

	addRef := func(ref FileRef) {
		id := strings.TrimSpace(ref.FileKey) + "\n" + strings.TrimSpace(ref.FileName)
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		refs = append(refs, ref)
	}

	for _, ref := range c.Files {
		addRef(ref)
	}
	for _, m := range c.Messages {
		for _, ref := range m.Files {
			addRef(ref)
		}
	}

	return refs
}

// buildFolderPaths maps folder IDs to output directories under outDir.
// Names are sanitized, and duplicate names get a numeric suffix.
func buildFolderPaths(outDir string, folders []Folder) map[string]string {
	folderPaths := make(map[string]string, len(folders))
	usedPaths := make(map[string]int, len(folders))

	for _, folder := range folders {
		id := strings.TrimSpace(folder.ID)
		if id == "" {
			continue
		}

		name := sanitizeFilename(folder.Name)
		if name == "" {
			name = "folder"
		}

		pathName := name
		if count, ok := usedPaths[pathName]; ok {
			count++
			usedPaths[pathName] = count
			pathName = fmt.Sprintf("%s_%d", name, count)
		} else {
			usedPaths[pathName] = 1
		}

		folderPaths[id] = filepath.Join(outDir, pathName)
	}

	return folderPaths
}

// defaultConversationFolder is where conversations without a mapped folder go.
func defaultConversationFolder(outDir string) string {
	return filepath.Join(outDir, "Unfiled")
}

// conversationMarkdownPath returns a unique Markdown path in folderDir named
// after the conversation, falling back to a hash of its ID. usedNames tracks
// names already issued so duplicates (case-insensitive) get a numeric suffix.
func conversationMarkdownPath(folderDir string, c Conversation, index int, usedNames map[string]int) string {
	baseName := sanitizeFilename(c.Name)
	if baseName == "" {
		id := strings.TrimSpace(c.ID)
		baseName = fmt.Sprintf("%s_%d", shortHashFromString(id, 10), index+1)
	}

	if ext := filepath.Ext(baseName); strings.EqualFold(ext, ".md") {
		baseName = strings.TrimSuffix(baseName, ext)
	}
	if baseName == "" {
		baseName = fmt.Sprintf("conversation_%d", index+1)
	}

	keyPrefix := folderDir + "\x00"
	key := keyPrefix + strings.ToLower(baseName)
	count := usedNames[key] + 1
	for {
		candidate := baseName
		if count > 1 {
			candidate = fmt.Sprintf("%s_%d", baseName, count)
		}
		candidateKey := keyPrefix + strings.ToLower(candidate)
		if _, exists := usedNames[candidateKey]; !exists {
			usedNames[key] = count
			usedNames[candidateKey] = 1
			baseName = candidate
			break
		}
		count++
	}

	return filepath.Join(folderDir, baseName+".md")
}

// markdownLink formats a link to a file at the slash-separated relative path rel.
func markdownLink(text, rel string) string {
	segments := strings.Split(rel, "/")
	for i, segment := range segments {
		segments[i] = linkPathEscaper.Replace(url.PathEscape(segment))
	}
	return fmt.Sprintf("[%s](%s)", linkTextEscaper.Replace(text), strings.Join(segments, "/"))
}

// convertConversation writes the Markdown file and attachments for one
// conversation. It reports whether the Markdown file was written; an existing
// file is kept unless overwrite is set. Per-attachment failures are recorded in
// the Markdown and logged rather than returned.
func (cv *converter) convertConversation(index int, c Conversation) (bool, error) {
	folderDir := defaultConversationFolder(cv.outDir)
	if folderPath, ok := cv.folderPaths[stringValue(c.FolderID)]; ok {
		folderDir = folderPath
	}
	if err := os.MkdirAll(folderDir, 0o755); err != nil {
		return false, fmt.Errorf("create conversation folder %q: %w", folderDir, err)
	}

	title := strings.TrimSpace(c.Name)
	if title == "" {
		title = fmt.Sprintf("Conversation %d", index+1)
	}
	mdPath := conversationMarkdownPath(folderDir, c, index, cv.usedNames)

	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	if id := strings.TrimSpace(c.ID); id != "" {
		fmt.Fprintf(&b, "_id: `%s`_\n\n", id)
	}
	writeMessages(&b, c.Messages)

	if err := cv.writeAttachments(&b, mdPath, filepath.Join(folderDir, cv.assetsDir), collectConversationFiles(c)); err != nil {
		return false, err
	}

	written, err := writeFile(mdPath, []byte(b.String()), cv.overwrite)
	if err != nil {
		return false, fmt.Errorf("write markdown: %w", err)
	}
	return written, nil
}

// writeMessages renders the Messages section.
func writeMessages(b *strings.Builder, messages []Message) {
	b.WriteString("## Messages\n\n")
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role == "" {
			role = "message"
		}
		b.WriteString("### " + role + "\n\n")
		b.WriteString(strings.TrimSpace(m.Content) + "\n\n")
	}
}

// writeAttachments extracts each referenced attachment into assetsDir and
// renders the Attachments section. It returns an error only if assetsDir
// cannot be created.
func (cv *converter) writeAttachments(b *strings.Builder, mdPath, assetsDir string, refs []FileRef) error {
	if len(refs) == 0 {
		return nil
	}
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("create assets dir %q: %w", assetsDir, err)
	}

	b.WriteString("## Attachments\n\n")
	for idx, ref := range refs {
		cv.writeAttachment(b, mdPath, assetsDir, idx, ref)
	}
	return nil
}

// writeAttachment extracts one attachment and appends its Markdown list item,
// or a note explaining why it could not be extracted.
func (cv *converter) writeAttachment(b *strings.Builder, mdPath, assetsDir string, idx int, ref FileRef) {
	key := strings.TrimSpace(ref.FileKey)
	origName := strings.TrimSpace(ref.FileName)
	if origName == "" {
		origName = fmt.Sprintf("attachment_%d", idx+1)
	}

	fail := func(msg string, err error) {
		slog.Warn("attachment skipped", "name", origName, "reason", msg, "error", err)
		fmt.Fprintf(b, "- **%s**: _%s: %v_\n", origName, msg, err)
	}

	blob, ok := cv.fileByKey[key]
	if !ok {
		slog.Warn("attachment skipped", "name", origName, "reason", "missing content", "fileKey", key)
		fmt.Fprintf(b, "- **%s**: _missing content for fileKey `%s`_\n", origName, key)
		return
	}

	contentBytes, resolvedName, err := attachmentPayload(blob, origName, idx)
	if err != nil {
		fail("failed to load attachment payload", err)
		return
	}

	safeName := sanitizeFilename(resolvedName)
	if safeName == "" {
		safeName = fmt.Sprintf("attachment_%d", idx+1)
	}

	dst, err := resolveAssetPath(assetsDir, safeName, contentBytes)
	if err != nil {
		fail("failed to resolve attachment path", err)
		return
	}
	if _, err := writeFile(dst, contentBytes, cv.overwrite); err != nil {
		fail("failed to write attachment", err)
		return
	}

	rel, err := filepath.Rel(filepath.Dir(mdPath), dst)
	if err != nil {
		fail("failed to link attachment", err)
		return
	}
	fmt.Fprintf(b, "- **%s**: %s\n", origName, markdownLink(filepath.Base(dst), filepath.ToSlash(rel)))
}
