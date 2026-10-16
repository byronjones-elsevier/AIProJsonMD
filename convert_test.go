package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestCollectConversationFiles(t *testing.T) {
	conv := Conversation{
		Files: []FileRef{
			{FileKey: "k1", FileName: "a.txt"},
			{FileKey: " k1 ", FileName: " a.txt "},
		},
		Messages: []Message{
			{Files: []FileRef{{FileKey: "k1", FileName: "a.txt"}, {FileKey: "k2", FileName: "b.txt"}}},
			{Files: []FileRef{{FileKey: "k2", FileName: "other-name.txt"}}},
		},
	}

	got := collectConversationFiles(conv)

	wantKeys := []string{"k1", "k2", "k2"}
	if len(got) != len(wantKeys) {
		t.Fatalf("got %d refs, want %d: %+v", len(got), len(wantKeys), got)
	}
	for i, ref := range got {
		if ref.FileKey != wantKeys[i] {
			t.Fatalf("ref %d key = %q, want %q", i, ref.FileKey, wantKeys[i])
		}
	}
}

func TestCollectConversationFilesEmpty(t *testing.T) {
	if got := collectConversationFiles(Conversation{}); len(got) != 0 {
		t.Fatalf("expected no refs, got %+v", got)
	}
}

func TestBuildFolderPaths(t *testing.T) {
	folders := []Folder{
		{ID: "1", Name: "Project A"},
		{ID: "2", Name: "Project A"},
		{ID: "3", Name: "Project A"},
		{ID: "4", Name: "  "},
		{ID: "5", Name: "a/b"},
		{ID: " ", Name: "ignored"},
	}

	got := buildFolderPaths("out", folders)

	want := map[string]string{
		"1": filepath.Join("out", "Project A"),
		"2": filepath.Join("out", "Project A_2"),
		"3": filepath.Join("out", "Project A_3"),
		"4": filepath.Join("out", "folder"),
		"5": filepath.Join("out", "a_b"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReservedWindowsNamesAreEscapedInPaths(t *testing.T) {
	path := conversationMarkdownPath("out", Conversation{Name: "CON"}, 0, map[string]int{})
	if want := filepath.Join("out", "_CON.md"); path != want {
		t.Fatalf("conversation path = %q, want %q", path, want)
	}

	folders := buildFolderPaths("out", []Folder{{ID: "1", Name: "nul"}})
	if want := filepath.Join("out", "_nul"); folders["1"] != want {
		t.Fatalf("folder path = %q, want %q", folders["1"], want)
	}
}

func TestMarkdownLink(t *testing.T) {
	tests := []struct {
		name string
		text string
		rel  string
		want string
	}{
		{"plain", "notes.txt", "assets/notes.txt", "[notes.txt](assets/notes.txt)"},
		{"spaces", "my notes.txt", "assets/my notes.txt", "[my notes.txt](assets/my%20notes.txt)"},
		{"parentheses", "a (1).txt", "assets/a (1).txt", "[a (1).txt](assets/a%20%281%29.txt)"},
		{"brackets in text", "a[1].txt", "assets/a1.txt", `[a\[1\].txt](assets/a1.txt)`},
		{"folder with spaces", "x.txt", "../My Folder/assets/x.txt", "[x.txt](../My%20Folder/assets/x.txt)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := markdownLink(tt.text, tt.rel); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunConvertsExport(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	outDir := filepath.Join(dir, "out")
	writeExport(t, inPath, Export{
		Version: 1,
		Folders: []Folder{{ID: "f1", Name: "Project A", Type: "chat"}},
		History: []Conversation{
			{
				ID:       "c1",
				Name:     "First: Chat",
				FolderID: "f1",
				Messages: []Message{{
					Role:    "user",
					Content: " hello ",
					Files:   []FileRef{{FileKey: "k1", FileName: "my notes (1).txt"}},
				}},
			},
			{
				ID:       "c2",
				FolderID: nil,
				Messages: []Message{{
					Role:  "assistant",
					Files: []FileRef{{FileKey: "gone", FileName: "lost.txt"}},
				}},
			},
			{
				ID:       "c3",
				Name:     "First: Chat",
				FolderID: "f1",
				Messages: []Message{{Role: "user", Content: "again", Files: []FileRef{{FileKey: "k3", FileName: "pic"}}}},
			},
		},
		Files: []FileBlob{
			{FileKey: "k1", FileName: "my notes (1).txt", Content: "note body"},
			{FileKey: "k3", FileName: "pic", FileType: "image/png", ImageBase64: "data:image/png;base64," + testPNGBase64},
		},
	})

	var stdout bytes.Buffer
	if err := run([]string{"-in", inPath, "-out", outDir}, &stdout); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Wrote 3 of 3 markdown files") {
		t.Fatalf("unexpected summary: %q", stdout.String())
	}

	first := readFile(t, filepath.Join(outDir, "Project A", "First_ Chat.md"))
	for _, want := range []string{
		"# First: Chat",
		"_id: `c1`_",
		"### user\n\nhello\n",
		"## Attachments",
		"- **my notes (1).txt**: [my notes (1).txt](assets/my%20notes%20%281%29.txt)",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("first conversation missing %q in:\n%s", want, first)
		}
	}
	if got := readFile(t, filepath.Join(outDir, "Project A", "assets", "my notes (1).txt")); got != "note body" {
		t.Errorf("attachment content = %q", got)
	}

	second := readFile(t, filepath.Join(outDir, "Project A", "First_ Chat_2.md"))
	if !strings.Contains(second, "[pic.png](assets/pic.png)") {
		t.Errorf("duplicate-name conversation missing image link in:\n%s", second)
	}
	if _, err := os.Stat(filepath.Join(outDir, "Project A", "assets", "pic.png")); err != nil {
		t.Errorf("image attachment not written: %v", err)
	}

	unfiled := readFile(t, filepath.Join(outDir, "Unfiled", shortHashFromString("c2", 10)+"_2.md"))
	for _, want := range []string{"# Conversation 2", "### assistant", "missing content for fileKey `gone`"} {
		if !strings.Contains(unfiled, want) {
			t.Errorf("unfiled conversation missing %q in:\n%s", want, unfiled)
		}
	}
}

func TestRunCreatesPrivateOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}

	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	outDir := filepath.Join(dir, "out")
	writeExport(t, inPath, Export{
		History: []Conversation{{
			ID:       "c1",
			Name:     "Private",
			Messages: []Message{{Role: "user", Files: []FileRef{{FileKey: "k1", FileName: "a.txt"}}}},
		}},
		Files: []FileBlob{{FileKey: "k1", FileName: "a.txt", Content: "secret"}},
	})

	if err := run([]string{"-in", inPath, "-out", outDir}, &bytes.Buffer{}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	tests := []struct {
		path string
		want fs.FileMode
	}{
		{outDir, 0o700},
		{filepath.Join(outDir, "Unfiled"), 0o700},
		{filepath.Join(outDir, "Unfiled", "assets"), 0o700},
		{filepath.Join(outDir, "Unfiled", "Private.md"), 0o600},
		{filepath.Join(outDir, "Unfiled", "assets", "a.txt"), 0o600},
	}
	for _, tt := range tests {
		info, err := os.Stat(tt.path)
		if err != nil {
			t.Fatalf("stat %q: %v", tt.path, err)
		}
		if got := info.Mode().Perm(); got != tt.want {
			t.Errorf("%s mode = %o, want %o", tt.path, got, tt.want)
		}
	}
}

func TestRunOverwriteBehavior(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	outDir := filepath.Join(dir, "out")
	writeExport(t, inPath, Export{
		History: []Conversation{
			{ID: "c1", Name: "One", Messages: []Message{{Role: "user", Content: "v1"}}},
			{ID: "c2", Name: "Two", Messages: []Message{{Role: "user", Content: "v1"}}},
		},
	})

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"first run writes everything", []string{"-in", inPath, "-out", outDir}, "Wrote 2 of 2"},
		{"rerun keeps existing files", []string{"-in", inPath, "-out", outDir}, "Wrote 0 of 2"},
		{"overwrite rewrites them", []string{"-in", inPath, "-out", outDir, "-overwrite"}, "Wrote 2 of 2"},
	}

	for _, tt := range tests {
		var stdout bytes.Buffer
		if err := run(tt.args, &stdout); err != nil {
			t.Fatalf("%s: run failed: %v", tt.name, err)
		}
		if !strings.Contains(stdout.String(), tt.want) {
			t.Fatalf("%s: summary %q does not contain %q", tt.name, stdout.String(), tt.want)
		}
	}
}

func TestRunErrors(t *testing.T) {
	dir := t.TempDir()
	badJSON := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badJSON, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	emptyHistory := filepath.Join(dir, "empty.json")
	writeExport(t, emptyHistory, Export{Version: 1})

	tests := []struct {
		name    string
		args    []string
		wantErr error
		anyErr  bool
	}{
		{"missing -in", nil, ErrMissingInput, false},
		{"blank -in", []string{"-in", "  "}, ErrMissingInput, false},
		{"unreadable input", []string{"-in", filepath.Join(dir, "nope.json")}, fs.ErrNotExist, false},
		{"invalid JSON", []string{"-in", badJSON}, nil, true},
		{"no history", []string{"-in", emptyHistory}, ErrNoHistory, false},
		{"assets escapes folder", []string{"-in", emptyHistory, "-assets", "../x"}, ErrInvalidAssetsDir, false},
		{"assets empty", []string{"-in", emptyHistory, "-assets", ""}, ErrInvalidAssetsDir, false},
		{"unknown flag", []string{"-bogus"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args, &bytes.Buffer{})
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func writeExport(t *testing.T, path string, export Export) {
	t.Helper()
	data, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal export: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write export: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return string(data)
}
