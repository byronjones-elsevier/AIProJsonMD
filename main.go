package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Export struct {
	Version int            `json:"version"`
	History []Conversation `json:"history"`
	Folders []Folder       `json:"folders"`
	Files   []FileBlob     `json:"files"`
}

type Conversation struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Messages []Message `json:"messages"`
	Files    []FileRef `json:"files"`
	FolderID any       `json:"folderId"`
}

type Folder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Message struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Files   []FileRef `json:"files"`
}

type FileRef struct {
	FileKey  string `json:"fileKey"`
	FileName string `json:"fileName"`
	FileSize any    `json:"fileSize"`
}

type FileBlob struct {
	FileKey        string `json:"fileKey"`
	FileName       string `json:"fileName"`
	FileSize       any    `json:"fileSize"`
	ConversationID string `json:"conversationId"`
	Content        string `json:"content"`
	FileType       any    `json:"fileType"`
	ImageBase64    any    `json:"imageBase64"`
}

// shortHashFromString computes the SHA-256 hash of the input string and returns the first n characters of its hexadecimal representation.
func shortHashFromString(s string, n int) string {
	sum := sha256.Sum256([]byte(s))
	full := hex.EncodeToString(sum[:])
	if n > len(full) {
		n = len(full)
	}
	return full[:n]
}

// sanitizeFilename replaces invalid characters in a filename with underscores and trims leading/trailing dots.
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	s = re.ReplaceAllString(s, "_")
	s = strings.Trim(s, ".")
	return s
}

// writeFile writes data to path, optionally overwriting an existing file.
func writeFile(path string, data []byte, overwrite bool) error {
	if overwrite {
		return os.WriteFile(path, data, 0644)
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// stringValue attempts to convert an interface{} value to a string, returning an empty string if the conversion is not possible.
func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

// mimeExtension returns the file extension corresponding to a given MIME type.
func mimeExtension(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "image/svg+xml":
		return ".svg"
	default:
		return ""
	}
}

// detectImageExtension attempts to determine the appropriate file extension for image data based on its MIME type or content.
func detectImageExtension(data []byte, fileType any) string {
	if ext := mimeExtension(stringValue(fileType)); ext != "" {
		return ext
	}

	return mimeExtension(http.DetectContentType(data))
}

// decodeBase64Payload decodes a base64-encoded string, handling potential data URI prefixes and whitespace.
func decodeBase64Payload(encoded string) ([]byte, error) {
	value := strings.TrimSpace(encoded)
	if value == "" {
		return nil, fmt.Errorf("empty base64 payload")
	}

	// Handle data URI prefix if present
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		value = value[comma+1:]
	}

	// Remove any whitespace or line breaks that may be present in the base64 string
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")

	// Attempt to decode using standard base64 encoding
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}

	// If standard decoding fails, attempt to decode using raw standard base64 encoding
	decoded, rawErr := base64.RawStdEncoding.DecodeString(value)
	if rawErr == nil {
		return decoded, nil
	}

	return nil, err
}

// attachmentPayload retrieves the content of a file blob, either from its direct content or by decoding its base64 image representation. It returns the decoded bytes, a resolved filename, and any error encountered during the process.
func attachmentPayload(blob FileBlob, origName string, idx int) ([]byte, string, error) {
	if content := blob.Content; content != "" {
		return []byte(content), origName, nil
	}

	// If the content is not directly available, attempt to decode the base64 image representation
	imageBase64 := stringValue(blob.ImageBase64)
	if imageBase64 == "" {
		return nil, origName, fmt.Errorf("no attachment payload found")
	}

	// Decode the base64 payload to obtain the binary data
	decoded, err := decodeBase64Payload(imageBase64)
	if err != nil {
		return nil, origName, fmt.Errorf("failed to decode imageBase64: %w", err)
	}

	// Determine the filename to use for the attachment, defaulting to "attachment_X" if the original name is empty
	name := origName
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("attachment_%d", idx+1)
	}

	// If the original name does not have an extension, attempt to detect the appropriate image extension based on the decoded data and file type
	if filepath.Ext(name) == "" {
		if ext := detectImageExtension(decoded, blob.FileType); ext != "" {
			name += ext
		}
	}

	return decoded, name, nil
}

// resolveAssetPath generates a unique file path for an asset in the specified directory. If a file with the same name already exists, it appends an index to the filename to avoid overwriting existing files. If a file with the same content already exists, it returns the existing file path.
func resolveAssetPath(dir string, safeName string, data []byte) (string, error) {
	baseName := strings.TrimSuffix(safeName, filepath.Ext(safeName))
	ext := filepath.Ext(safeName)

	// Ensure the directory exists
	for index := 0; ; index++ {
		name := safeName
		if index > 0 {
			name = fmt.Sprintf("%s_%d%s", baseName, index+1, ext)
		}

		// Check if a file with the same name already exists in the directory
		candidate := filepath.Join(dir, name)
		existing, err := os.ReadFile(candidate)
		if err == nil {
			if bytes.Equal(existing, data) {
				return candidate, nil
			}
			continue
		}

		// If the file does not exist, return the candidate path for writing
		if os.IsNotExist(err) {
			return candidate, nil
		}

		return "", err
	}
}

// collectConversationFiles aggregates all unique file references from a conversation, including those directly associated with the conversation and those attached to its messages. It returns a slice of FileRef containing the unique file references.
func collectConversationFiles(c Conversation) []FileRef {
	refs := make([]FileRef, 0, len(c.Files))
	seen := make(map[string]struct{})

	// Helper function to add a file reference to the collection if it hasn't been seen before
	addRef := func(ref FileRef) {
		key := strings.TrimSpace(ref.FileKey)
		name := strings.TrimSpace(ref.FileName)
		id := key + "\n" + name
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		refs = append(refs, ref)
	}

	// Collect file references directly associated with the conversation
	for _, ref := range c.Files {
		addRef(ref)
	}

	// Collect file references from each message in the conversation
	for _, m := range c.Messages {
		for _, ref := range m.Files {
			addRef(ref)
		}
	}

	return refs
}

// folderIDString converts a folder ID of any type to a string representation, returning an empty string if the conversion is not possible.
func folderIDString(value any) string {
	return stringValue(value)
}

// buildFolderPaths constructs a mapping of folder IDs to their corresponding output paths based on the provided output directory and folder information. It ensures that folder names are sanitized and unique within the output directory.
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

		// Ensure unique folder names by appending an index if a folder with the same name already exists
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

// defaultConversationFolder returns the default folder path for conversations that do not belong to any specific folder. It appends "Unfiled" to the provided output directory.
func defaultConversationFolder(outDir string) string {
	return filepath.Join(outDir, "Unfiled")
}

// conversationMarkdownPath returns the markdown path for a conversation, preferring its name when available.
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

	key := folderDir + "\x00" + strings.ToLower(baseName)
	if count, ok := usedNames[key]; ok {
		count++
		usedNames[key] = count
		baseName = fmt.Sprintf("%s_%d", baseName, count)
	} else {
		usedNames[key] = 1
	}

	return filepath.Join(folderDir, baseName+".md")
}

// main is the entry point of the program. It processes command-line flags, reads the input JSON file, and generates markdown files for each conversation along with their attachments.
func main() {

	//Collect command-line flags and arguments
	inPath := flag.String("in", "", "Path to input JSON file")
	outDir := flag.String("out", "out_markdown", "Output directory for markdown files")
	assetsDir := flag.String("assets", "assets", "Per-folder assets directory name")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing markdown and attachment files")
	flag.Parse()

	// Validate input path and provide usage instructions if not provided
	if strings.TrimSpace(*inPath) == "" {
		fmt.Println("Usage: go run . -in export.json -out out_markdown -assets assets -overwrite")
		os.Exit(1)
	}

	// Read and parse the input JSON file
	data, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Println("Failed to read input:", err)
		os.Exit(1)
	}

	// Unmarshal the JSON data into the Export struct
	var export Export
	if err := json.Unmarshal(data, &export); err != nil {
		fmt.Println("Failed to parse JSON:", err)
		os.Exit(1)
	}

	// Check if there are any history entries in the export
	if len(export.History) == 0 {
		fmt.Println("No history entries found.")
		os.Exit(1)
	}

	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Println("Failed to create markdown output directory:", err)
		os.Exit(1)
	}

	// Build a map of folder IDs to their corresponding paths
	folderPaths := buildFolderPaths(*outDir, export.Folders)

	// Index end-of-JSON files by fileKey
	fileByKey := make(map[string]FileBlob, len(export.Files))
	for _, fb := range export.Files {
		key := strings.TrimSpace(fb.FileKey)
		if key != "" {
			fileByKey[key] = fb
		}
	}

	// Process each conversation in the history
	usedMarkdownNames := make(map[string]int, len(export.History))
	for i, c := range export.History {
		id := strings.TrimSpace(c.ID)
		fileRefs := collectConversationFiles(c)
		folderDir := defaultConversationFolder(*outDir)
		if folderPath, ok := folderPaths[folderIDString(c.FolderID)]; ok {
			folderDir = folderPath
		}

		// Create the folder for the conversation if it doesn't exist
		if err := os.MkdirAll(folderDir, 0755); err != nil {
			fmt.Println("Failed to create conversation folder:", err)
			os.Exit(1)
		}

		// Determine the title for the conversation, defaulting to "Conversation X" if not provided
		title := strings.TrimSpace(c.Name)
		if title == "" {
			title = fmt.Sprintf("Conversation %d", i+1)
		}

		// Construct the markdown file path for the conversation
		mdPath := conversationMarkdownPath(folderDir, c, i, usedMarkdownNames)

		// Build the markdown content for the conversation
		var b strings.Builder
		b.WriteString("# ")
		b.WriteString(title)
		b.WriteString("\n\n")
		if id != "" {
			b.WriteString(fmt.Sprintf("_id: `%s`_\n\n", id))
		}

		// Write the messages section of the markdown
		b.WriteString("## Messages\n\n")
		for _, m := range c.Messages {
			role := strings.TrimSpace(m.Role)
			if role == "" {
				role = "message"
			}
			b.WriteString("### ")
			b.WriteString(role)
			b.WriteString("\n\n")
			b.WriteString(strings.TrimSpace(m.Content))
			b.WriteString("\n\n")
		}

		// Write the attachments section of the markdown, if any
		assetsForConvDir := filepath.Join(folderDir, *assetsDir)
		if len(fileRefs) > 0 {
			if err := os.MkdirAll(assetsForConvDir, 0755); err != nil {
				fmt.Println("Failed to create assets dir:", err)
				os.Exit(1)
			}

			b.WriteString("## Attachments\n\n")

			// Process each file reference and handle attachments
			for idx, ref := range fileRefs {
				key := strings.TrimSpace(ref.FileKey)
				origName := strings.TrimSpace(ref.FileName)

				if origName == "" {
					origName = fmt.Sprintf("attachment_%d", idx+1)
				}

				// Look up the file blob by its key
				blob, ok := fileByKey[key]
				if !ok {
					b.WriteString(fmt.Sprintf("- **%s**: _missing content for fileKey `%s`_\n", origName, key))
					continue
				}

				// Get the attachment payload and resolved name
				contentBytes, resolvedName, err := attachmentPayload(blob, origName, idx)
				if err != nil {
					b.WriteString(fmt.Sprintf("- **%s**: _failed to load attachment payload: %v_\n", origName, err))
					continue
				}

				// Sanitize the filename to ensure it's safe for the filesystem
				safeName := sanitizeFilename(resolvedName)
				if safeName == "" {
					safeName = fmt.Sprintf("attachment_%d", idx+1)
				}

				// Resolve the asset path, ensuring no conflicts with existing files
				dst, err := resolveAssetPath(assetsForConvDir, safeName, contentBytes)
				if err != nil {
					b.WriteString(fmt.Sprintf("- **%s**: _failed to resolve attachment path: %v_\n", origName, err))
					continue
				}

				// Write attachment file, honoring the overwrite option.
				if err := writeFile(dst, contentBytes, *overwrite); err != nil {
					b.WriteString(fmt.Sprintf("- **%s**: _failed to write attachment: %v_\n", origName, err))
					continue
				}

				// Write the attachment link in the markdown file
				rel, _ := filepath.Rel(filepath.Dir(mdPath), dst)
				rel = filepath.ToSlash(rel)
				b.WriteString(fmt.Sprintf("- **%s**: [%s](%s)\n", origName, filepath.Base(dst), rel))
			}
		}

		// Write markdown file, honoring the overwrite option.
		if err := writeFile(mdPath, []byte(b.String()), *overwrite); err != nil {
			fmt.Printf("Failed writing markdown for conversation %d: %v\n", i, err)
			os.Exit(1)
		}
	}

	// Print completion message with the number of markdown files written and the output directory
	fmt.Printf("Done. Wrote %d markdown files to %s\n", len(export.History), *outDir)
	fmt.Printf("Attachments written under per-folder %s directories\n", *assetsDir)
}
