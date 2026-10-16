package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// ErrEmptyBase64 is returned when a base64 payload has no content.
	ErrEmptyBase64 = errors.New("empty base64 payload")
	// ErrNoPayload is returned when an attachment has neither text content nor image data.
	ErrNoPayload = errors.New("no attachment payload found")
	// ErrInvalidAssetsDir is returned when -assets is not a single directory name.
	ErrInvalidAssetsDir = errors.New("invalid -assets value: must be a single directory name")
)

const (
	// filePerm and dirPerm keep exported chat history private to the current user.
	// They apply only to files and directories this tool creates.
	filePerm fs.FileMode = 0o600
	dirPerm  fs.FileMode = 0o700

	// maxNameBytes caps a sanitized file or folder name, leaving room for the
	// ".md" extension and "_N" collision suffixes under the common 255-byte limit.
	maxNameBytes = 200
)

// invalidFilenameChars matches characters that are unsafe in file names on common filesystems.
var invalidFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

// validateAssetsDir reports whether name is usable as the per-folder assets
// directory: a single, non-empty path element that cannot escape its folder.
func validateAssetsDir(name string) error {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) || strings.ContainsFunc(name, unicode.IsControl) {
		return fmt.Errorf("%w: %q", ErrInvalidAssetsDir, name)
	}
	return nil
}

// truncateName shortens name to at most limit bytes without splitting a UTF-8
// character, preserving a short file extension.
func truncateName(name string, limit int) string {
	if len(name) <= limit {
		return name
	}

	ext := filepath.Ext(name)
	if len(ext) > 16 || len(ext) >= limit {
		ext = ""
	}
	base := strings.TrimSuffix(name, ext)

	cut := limit - len(ext)
	for cut > 0 && !utf8.RuneStart(base[cut]) {
		cut--
	}
	return strings.TrimRight(base[:cut], ". ") + ext
}

// shortHashFromString returns the first n hex characters of the SHA-256 of s.
func shortHashFromString(s string, n int) string {
	sum := sha256.Sum256([]byte(s))
	full := hex.EncodeToString(sum[:])
	if n > len(full) {
		n = len(full)
	}
	return full[:n]
}

// sanitizeFilename replaces filesystem-unsafe characters with underscores and trims surrounding dots.
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = invalidFilenameChars.ReplaceAllString(s, "_")
	s = strings.TrimRight(strings.TrimLeft(s, "."), ". ")
	return truncateName(guardReservedName(s), maxNameBytes)
}

// windowsReservedNames are device names Windows refuses as a file name, with
// or without an extension (for example "CON" and "con.txt").
var windowsReservedNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {},
	"COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {},
	"LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

// guardReservedName prefixes an underscore when the part of name before its
// first dot is a Windows reserved device name, so the output can be copied to
// or created on Windows.
func guardReservedName(name string) string {
	stem, _, _ := strings.Cut(name, ".")
	if _, reserved := windowsReservedNames[strings.ToUpper(strings.TrimRight(stem, " "))]; reserved {
		return "_" + name
	}
	return name
}

// writeFile writes data to path and reports whether the file was written.
// Without overwrite, an existing file is left untouched and written is false.
func writeFile(path string, data []byte, overwrite bool) (written bool, err error) {
	if overwrite {
		if err := os.WriteFile(path, data, filePerm); err != nil {
			return false, fmt.Errorf("write %q: %w", path, err)
		}
		return true, nil
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create %q: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		return false, fmt.Errorf("write %q: %w", path, errors.Join(err, f.Close()))
	}
	if err := f.Close(); err != nil {
		return false, fmt.Errorf("close %q: %w", path, err)
	}
	return true, nil
}

// stringValue returns the trimmed string held by value, or "" for any other type.
func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// mimeExtension maps a supported image MIME type to its file extension, or "" if unknown.
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

// detectImageExtension prefers the declared fileType and falls back to sniffing the data.
func detectImageExtension(data []byte, fileType any) string {
	if ext := mimeExtension(stringValue(fileType)); ext != "" {
		return ext
	}
	return mimeExtension(http.DetectContentType(data))
}

// decodeBase64Payload decodes standard or unpadded base64, tolerating a data URI prefix and line breaks.
func decodeBase64Payload(encoded string) ([]byte, error) {
	value := strings.TrimSpace(encoded)
	if value == "" {
		return nil, ErrEmptyBase64
	}

	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		value = value[comma+1:]
	}
	value = strings.NewReplacer("\n", "", "\r", "").Replace(value)

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	decoded, rawErr := base64.RawStdEncoding.DecodeString(value)
	if rawErr == nil {
		return decoded, nil
	}
	return nil, errors.Join(err, rawErr)
}

// attachmentPayload returns the bytes and file name for a blob, preferring text
// content over decoded imageBase64. Image names without an extension get one
// inferred from the file type or the decoded bytes.
func attachmentPayload(blob FileBlob, origName string, idx int) ([]byte, string, error) {
	if blob.Content != "" {
		return []byte(blob.Content), origName, nil
	}

	imageBase64 := stringValue(blob.ImageBase64)
	if imageBase64 == "" {
		return nil, origName, ErrNoPayload
	}

	decoded, err := decodeBase64Payload(imageBase64)
	if err != nil {
		return nil, origName, fmt.Errorf("decode imageBase64: %w", err)
	}

	name := origName
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("attachment_%d", idx+1)
	}
	if filepath.Ext(name) == "" {
		name += detectImageExtension(decoded, blob.FileType)
	}

	return decoded, name, nil
}

// resolveAssetPath picks a path in dir for an asset named safeName. A file with
// identical content is reused; a name collision with different content gets a
// numeric suffix (name_2.ext, name_3.ext, ...).
func resolveAssetPath(dir string, safeName string, data []byte) (string, error) {
	ext := filepath.Ext(safeName)
	baseName := strings.TrimSuffix(safeName, ext)

	for index := 0; ; index++ {
		name := safeName
		if index > 0 {
			name = fmt.Sprintf("%s_%d%s", baseName, index+1, ext)
		}

		candidate := filepath.Join(dir, name)
		existing, err := os.ReadFile(candidate)
		if err == nil {
			if bytes.Equal(existing, data) {
				return candidate, nil
			}
			continue
		}
		if errors.Is(err, fs.ErrNotExist) {
			return candidate, nil
		}
		return "", fmt.Errorf("read %q: %w", candidate, err)
	}
}
