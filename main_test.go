package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFile_NoOverwritePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")

	if err := writeFile(path, []byte("first"), false); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}
	if err := writeFile(path, []byte("second"), false); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("expected existing file to be preserved, got %q", string(data))
	}
}

func TestWriteFile_OverwriteReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")

	if err := writeFile(path, []byte("first"), false); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}
	if err := writeFile(path, []byte("second"), true); err != nil {
		t.Fatalf("overwrite write failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("expected file to be overwritten, got %q", string(data))
	}
}

func TestAttachmentPayload_UsesImageBase64AndAddsExtension(t *testing.T) {
	pngBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO6m9S8AAAAASUVORK5CYII="
	blob := FileBlob{
		ImageBase64: "data:image/png;base64," + pngBase64,
		FileType:    "image/png",
	}

	data, name, err := attachmentPayload(blob, "photo", 0)
	if err != nil {
		t.Fatalf("attachmentPayload failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected decoded image bytes")
	}
	if filepath.Ext(name) != ".png" {
		t.Fatalf("expected .png extension, got %q", filepath.Ext(name))
	}
}

func TestDecodeBase64Payload_RawBase64(t *testing.T) {
	encoded := base64.RawStdEncoding.EncodeToString([]byte("hello"))
	decoded, err := decodeBase64Payload(encoded)
	if err != nil {
		t.Fatalf("decodeBase64Payload failed: %v", err)
	}
	if string(decoded) != "hello" {
		t.Fatalf("expected hello, got %q", string(decoded))
	}
}

func TestConversationMarkdownPath_UsesSanitizedName(t *testing.T) {
	usedNames := map[string]int{}
	path := conversationMarkdownPath("out", Conversation{Name: `My / Chat: Notes? v1.2`}, 0, usedNames)

	want := filepath.Join("out", "My _ Chat_ Notes_ v1.2.md")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestConversationMarkdownPath_DoesNotDuplicateMarkdownExtension(t *testing.T) {
	usedNames := map[string]int{}
	path := conversationMarkdownPath("out", Conversation{Name: "Already Markdown.md"}, 0, usedNames)

	want := filepath.Join("out", "Already Markdown.md")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestConversationMarkdownPath_FallsBackToHashForBlankName(t *testing.T) {
	usedNames := map[string]int{}
	c := Conversation{ID: "conversation-id"}
	path := conversationMarkdownPath("out", c, 2, usedNames)

	want := filepath.Join("out", shortHashFromString(c.ID, 10)+"_3.md")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestConversationMarkdownPath_DeduplicatesNamesPerFolder(t *testing.T) {
	usedNames := map[string]int{}
	first := conversationMarkdownPath("out", Conversation{Name: "Same Name"}, 0, usedNames)
	second := conversationMarkdownPath("out", Conversation{Name: "Same Name"}, 1, usedNames)
	third := conversationMarkdownPath(filepath.Join("out", "Nested"), Conversation{Name: "Same Name"}, 2, usedNames)

	if first != filepath.Join("out", "Same Name.md") {
		t.Fatalf("expected first name-based path, got %q", first)
	}
	if second != filepath.Join("out", "Same Name_2.md") {
		t.Fatalf("expected duplicate suffix, got %q", second)
	}
	if third != filepath.Join("out", "Nested", "Same Name.md") {
		t.Fatalf("expected folder-local naming, got %q", third)
	}
}
