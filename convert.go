package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// linkPathEscaper escapes the parentheses that url.PathEscape leaves intact,
// since they would end a Markdown link destination early.
var linkPathEscaper = strings.NewReplacer("(", "%28", ")", "%29")

// linkTextEscaper escapes brackets that would end Markdown link text early.
var linkTextEscaper = strings.NewReplacer("[", `\[`, "]", `\]`)

// unfiledName is the folder for conversations without a mapped export folder.
const unfiledName = "Unfiled"

// mainIndexName is the page at the top of the output directory.
const mainIndexName = "index.html"

// reservedConversationName is never used for a conversation file. Each folder
// has an index page with this base name, so a conversation titled "index" is
// written as "index_2" to keep the folder page from being replaced. The name is
// reserved for every format so Markdown and HTML names always match.
const reservedConversationName = "index"

// converter turns export conversations into Markdown and HTML files plus
// extracted attachments.
type converter struct {
	outDir      string
	assetsDir   string
	overwrite   bool
	format      outputFormat
	html        *htmlRenderer
	folderPaths map[string]string
	fileByKey   map[string]FileBlob
	usedNames   map[string]int
	index       []indexEntry
}

// newConverter indexes the export's folders and attachment blobs for lookup
// during conversion, and prepares the HTML renderer when HTML is requested.
func newConverter(exp Export, outDir, assetsDir string, overwrite bool, format outputFormat) (*converter, error) {
	fileByKey := make(map[string]FileBlob, len(exp.Files))
	for _, fb := range exp.Files {
		if key := strings.TrimSpace(fb.FileKey); key != "" {
			fileByKey[key] = fb
		}
	}

	cv := &converter{
		outDir:      outDir,
		assetsDir:   assetsDir,
		overwrite:   overwrite,
		format:      format,
		folderPaths: buildFolderPaths(outDir, exp.Folders),
		fileByKey:   fileByKey,
		usedNames:   make(map[string]int, len(exp.History)),
	}
	if format.writesHTML() {
		renderer, err := newHTMLRenderer()
		if err != nil {
			return nil, err
		}
		cv.html = renderer
	}
	return cv, nil
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
	// Names are compared case-insensitively so folders that differ only in case
	// do not share a directory on case-insensitive filesystems. The main index
	// file lives in the output directory next to the folders, so its name is taken.
	used := map[string]struct{}{strings.ToLower(mainIndexName): {}}

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
		for n := 2; ; n++ {
			if _, taken := used[strings.ToLower(pathName)]; !taken {
				break
			}
			pathName = fmt.Sprintf("%s_%d", name, n)
		}
		used[strings.ToLower(pathName)] = struct{}{}

		folderPaths[id] = filepath.Join(outDir, pathName)
	}

	return folderPaths
}

// defaultConversationFolder is where conversations without a mapped folder go.
func defaultConversationFolder(outDir string) string {
	return filepath.Join(outDir, unfiledName)
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
	reservedKey := keyPrefix + reservedConversationName
	if _, ok := usedNames[reservedKey]; !ok {
		usedNames[reservedKey] = 1
	}
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

// escapeLinkPath URL-escapes each segment of a slash-separated relative path,
// including the parentheses that would end a Markdown link destination early.
func escapeLinkPath(rel string) string {
	segments := strings.Split(rel, "/")
	for i, segment := range segments {
		segments[i] = linkPathEscaper.Replace(url.PathEscape(segment))
	}
	return strings.Join(segments, "/")
}

// markdownLink formats a link to a file at the slash-separated relative path rel.
func markdownLink(text, rel string) string {
	return fmt.Sprintf("[%s](%s)", linkTextEscaper.Replace(text), escapeLinkPath(rel))
}
