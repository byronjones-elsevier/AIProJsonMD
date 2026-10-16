package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO6m9S8AAAAASUVORK5CYII="

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"plain", "notes.txt", "notes.txt"},
		{"invalid characters", `a/b:c*d?e"f<g>h|i\j`, "a_b_c_d_e_f_g_h_i_j"},
		{"control characters", "a\x00b\x1fc", "a_b_c"},
		{"surrounding dots", "..hidden..", "hidden"},
		{"only dots", "..", ""},
		{"keeps inner spaces and parens", "my notes (1).txt", "my notes (1).txt"},
		{"trailing dot and space", "a .", "a"},
		{"reserved name", "CON", "_CON"},
		{"reserved name lowercase", "nul", "_nul"},
		{"reserved name with extension", "aux.txt", "_aux.txt"},
		{"reserved name with several dots", "com1.tar.gz", "_com1.tar.gz"},
		{"reserved device number", "LPT9", "_LPT9"},
		{"reserved name with trailing space before dot", "PRN .txt", "_PRN .txt"},
		{"COM0 is not reserved", "COM0", "COM0"},
		{"reserved name as prefix is fine", "console.txt", "console.txt"},
		{"reserved name inside is fine", "my CON notes", "my CON notes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFilename(tt.in); got != tt.want {
				t.Fatalf("sanitizeFilename(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateAssetsDir(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"default", "assets", false},
		{"with spaces", "my assets", false},
		{"dotted name", "assets.v2", false},
		{"empty", "", true},
		{"blank", "  ", true},
		{"current dir", ".", true},
		{"parent dir", "..", true},
		{"parent traversal", "../x", true},
		{"nested", "media/img", true},
		{"absolute", "/tmp/assets", true},
		{"backslash", `a\b`, true},
		{"control character", "a\x00b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAssetsDir(tt.in)
			if tt.wantErr && !errors.Is(err, ErrInvalidAssetsDir) {
				t.Fatalf("validateAssetsDir(%q) = %v, want ErrInvalidAssetsDir", tt.in, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateAssetsDir(%q) unexpected error: %v", tt.in, err)
			}
		})
	}
}

func TestTruncateName(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		limit int
		want  string
	}{
		{"short name unchanged", "notes.txt", 20, "notes.txt"},
		{"exactly at limit", "abcdefghij", 10, "abcdefghij"},
		{"extension preserved", "abcdefghijklmnop.png", 10, "abcdef.png"},
		{"no extension", "abcdefghijklmnop", 10, "abcdefghij"},
		{"trailing dot and space trimmed", "abcd. efghijklmnop", 6, "abcd"},
		{"long extension ignored", "ab." + strings.Repeat("x", 30), 10, "ab." + strings.Repeat("x", 7)},
		{"multibyte not split", strings.Repeat("é", 10), 5, "éé"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateName(tt.in, tt.limit)
			if got != tt.want {
				t.Fatalf("truncateName(%q, %d) = %q, want %q", tt.in, tt.limit, got, tt.want)
			}
			if len(got) > tt.limit {
				t.Fatalf("result is %d bytes, over limit %d", len(got), tt.limit)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("result %q is not valid UTF-8", got)
			}
		})
	}
}

func TestSanitizeFilenameCapsLength(t *testing.T) {
	got := sanitizeFilename(strings.Repeat("a", 500) + ".txt")
	if len(got) > maxNameBytes {
		t.Fatalf("name is %d bytes, over %d", len(got), maxNameBytes)
	}
	if !strings.HasSuffix(got, ".txt") {
		t.Fatalf("extension lost: %q", got)
	}
}

func TestShortHashFromString(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantLen int
	}{
		{"requested length", 10, 10},
		{"zero", 0, 0},
		{"clamped to full digest", 100, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortHashFromString("conversation-id", tt.n)
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
			if again := shortHashFromString("conversation-id", tt.n); again != got {
				t.Fatalf("hash is not deterministic: %q vs %q", got, again)
			}
		})
	}
}

func TestMimeExtension(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"image/png", ".png"},
		{"IMAGE/JPEG", ".jpg"},
		{" image/jpg ", ".jpg"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/bmp", ".bmp"},
		{"image/svg+xml", ".svg"},
		{"text/plain", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := mimeExtension(tt.in); got != tt.want {
				t.Fatalf("mimeExtension(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestStringValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string is trimmed", "  hi ", "hi"},
		{"nil", nil, ""},
		{"number", 42, ""},
		{"float", 1.5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringValue(tt.in); got != tt.want {
				t.Fatalf("stringValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDecodeBase64Payload(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
		anyErr  bool
	}{
		{name: "padded", in: "aGVsbG8=", want: "hello"},
		{name: "unpadded", in: "aGVsbG8", want: "hello"},
		{name: "data URI", in: "data:text/plain;base64,aGVsbG8=", want: "hello"},
		{name: "line breaks", in: "aGVs\r\nbG8=", want: "hello"},
		{name: "empty", in: "", wantErr: ErrEmptyBase64},
		{name: "whitespace only", in: " \n ", wantErr: ErrEmptyBase64},
		{name: "invalid", in: "!!!not base64!!!", anyErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBase64Payload(tt.in)
			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			case tt.anyErr:
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(got) != tt.want {
					t.Fatalf("got %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestAttachmentPayload(t *testing.T) {
	tests := []struct {
		name     string
		blob     FileBlob
		origName string
		idx      int
		wantName string
		wantData string
		wantErr  error
		anyErr   bool
	}{
		{
			name:     "text content keeps name",
			blob:     FileBlob{Content: "hello"},
			origName: "notes.txt",
			wantName: "notes.txt",
			wantData: "hello",
		},
		{
			name:     "content wins over image data",
			blob:     FileBlob{Content: "hello", ImageBase64: testPNGBase64},
			origName: "notes.txt",
			wantName: "notes.txt",
			wantData: "hello",
		},
		{
			name:     "extension sniffed from bytes",
			blob:     FileBlob{ImageBase64: testPNGBase64},
			origName: "photo",
			wantName: "photo.png",
		},
		{
			name:     "existing extension kept",
			blob:     FileBlob{ImageBase64: testPNGBase64, FileType: "image/png"},
			origName: "photo.jpeg",
			wantName: "photo.jpeg",
		},
		{
			name:     "blank name generated from index",
			blob:     FileBlob{ImageBase64: testPNGBase64, FileType: "image/png"},
			origName: "  ",
			idx:      2,
			wantName: "attachment_3.png",
		},
		{
			name:     "no payload",
			blob:     FileBlob{},
			origName: "empty.txt",
			wantErr:  ErrNoPayload,
		},
		{
			name:     "invalid base64",
			blob:     FileBlob{ImageBase64: "!!!"},
			origName: "bad.png",
			anyErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, name, err := attachmentPayload(tt.blob, tt.origName, tt.idx)
			if tt.wantErr != nil || tt.anyErr {
				if err == nil {
					t.Fatalf("expected an error, got name %q", name)
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tt.wantName {
				t.Fatalf("name = %q, want %q", name, tt.wantName)
			}
			if tt.wantData != "" && string(data) != tt.wantData {
				t.Fatalf("data = %q, want %q", data, tt.wantData)
			}
			if len(data) == 0 {
				t.Fatalf("expected non-empty data")
			}
		})
	}
}

func TestResolveAssetPath(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("setup write failed: %v", err)
		}
	}

	t.Run("free name is used as is", func(t *testing.T) {
		got, err := resolveAssetPath(dir, "a.txt", []byte("one"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := filepath.Join(dir, "a.txt"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	write("a.txt", "one")

	t.Run("identical content reuses the file", func(t *testing.T) {
		got, err := resolveAssetPath(dir, "a.txt", []byte("one"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := filepath.Join(dir, "a.txt"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("different content gets a suffix", func(t *testing.T) {
		got, err := resolveAssetPath(dir, "a.txt", []byte("two"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := filepath.Join(dir, "a_2.txt"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	write("a_2.txt", "two")

	t.Run("suffix keeps counting past taken names", func(t *testing.T) {
		got, err := resolveAssetPath(dir, "a.txt", []byte("three"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := filepath.Join(dir, "a_3.txt"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("suffixed name with identical content is reused", func(t *testing.T) {
		got, err := resolveAssetPath(dir, "a.txt", []byte("two"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := filepath.Join(dir, "a_2.txt"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

func TestWriteFileReportsWhetherItWrote(t *testing.T) {
	tests := []struct {
		name        string
		existing    string
		hasExisting bool
		overwrite   bool
		wantWritten bool
		wantContent string
	}{
		{name: "new file", overwrite: false, wantWritten: true, wantContent: "new"},
		{name: "existing kept", existing: "old", hasExisting: true, overwrite: false, wantWritten: false, wantContent: "old"},
		{name: "existing replaced", existing: "old", hasExisting: true, overwrite: true, wantWritten: true, wantContent: "new"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.txt")
			if tt.hasExisting {
				if err := os.WriteFile(path, []byte(tt.existing), 0o600); err != nil {
					t.Fatalf("setup write failed: %v", err)
				}
			}

			written, err := writeFile(path, []byte("new"), tt.overwrite)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if written != tt.wantWritten {
				t.Fatalf("written = %v, want %v", written, tt.wantWritten)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read failed: %v", err)
			}
			if string(data) != tt.wantContent {
				t.Fatalf("content = %q, want %q", data, tt.wantContent)
			}
		})
	}
}

func TestWriteFileMissingDirectoryFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "f.txt")
	for _, overwrite := range []bool{false, true} {
		if written, err := writeFile(path, []byte("x"), overwrite); err == nil || written {
			t.Fatalf("overwrite=%v: got written=%v err=%v, want an error", overwrite, written, err)
		}
	}
}
