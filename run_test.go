package main

import (
	"bytes"
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// snapshotTree returns the contents of every regular file under root, keyed by
// its slash-separated path relative to root.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	var rels []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	files := make(map[string]string, len(rels))
	for _, rel := range rels {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		files[filepath.ToSlash(rel)] = string(data)
	}
	return files
}

// TestRunIsDeterministic checks REQ-OUT-4: two runs on the same input produce
// byte-identical output trees.
func TestRunIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	writeExport(t, inPath, htmlSampleExport())

	for _, format := range []string{"md", "html", "both"} {
		first := filepath.Join(dir, "first-"+format)
		second := filepath.Join(dir, "second-"+format)
		for _, out := range []string{first, second} {
			if err := run([]string{"-in", inPath, "-out", out, "-format", format}, &bytes.Buffer{}); err != nil {
				t.Fatalf("run %s: %v", format, err)
			}
		}

		a, b := snapshotTree(t, first), snapshotTree(t, second)
		if len(a) == 0 {
			t.Fatalf("%s run wrote no files", format)
		}
		if len(a) != len(b) {
			t.Fatalf("%s: runs wrote %d and %d files", format, len(a), len(b))
		}
		for name, want := range a {
			if got, ok := b[name]; !ok || got != want {
				t.Errorf("%s: %s differs between two runs", format, name)
			}
		}
	}
}

// TestRunHelpReturnsErrHelp checks REQ-CLI-5: -h and -help return the help
// error, which main treats as success (exit status 0, nothing logged).
func TestRunHelpReturnsErrHelp(t *testing.T) {
	for _, arg := range []string{"-h", "-help", "--help"} {
		err := run([]string{arg}, &bytes.Buffer{})
		if !errors.Is(err, flag.ErrHelp) {
			t.Errorf("run(%q) = %v, want flag.ErrHelp", arg, err)
		}
	}
}

// TestAttachmentWarningIsLogged checks the warning line format in section 8
// (REQ-CLI-6, REQ-ATT-4): the default logger's date and time, the level, the
// message, and the fields in order, with spaces in a value quoted.
func TestAttachmentWarningIsLogged(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	writeExport(t, inPath, Export{
		History: []Conversation{{
			ID:   "c1",
			Name: "Warn",
			Messages: []Message{{Role: "user", Files: []FileRef{
				{FileKey: "gone", FileName: "lost.txt"},
				{FileKey: "empty", FileName: "empty.txt"},
			}}},
		}},
		Files: []FileBlob{{FileKey: "empty", FileName: "empty.txt"}},
	})
	if err := run([]string{"-in", inPath, "-out", filepath.Join(dir, "out")}, &bytes.Buffer{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	prefix := `\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} WARN attachment skipped `
	wants := []string{
		prefix + `name=lost\.txt reason="missing content" fileKey=gone`,
		prefix + `name=empty\.txt reason="failed to load attachment payload" error="no attachment payload found"`,
	}
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != len(wants) {
		t.Fatalf("logged %d lines, want %d:\n%s", len(lines), len(wants), logs.String())
	}
	for i, want := range wants {
		if !regexp.MustCompile("^" + want + "$").MatchString(lines[i]) {
			t.Errorf("log line %d = %q, want to match %q", i, lines[i], want)
		}
	}
}

// nonTestSource returns the text of the program's Go source files, excluding tests.
func nonTestSource(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	source := map[string]string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		source[f] = string(data)
	}
	if len(source) == 0 {
		t.Fatalf("found no Go source files")
	}
	return source
}

// TestProgramNeverDeletesFiles checks REQ-OUT-2: the program never removes,
// renames, or truncates files.
func TestProgramNeverDeletesFiles(t *testing.T) {
	forbidden := []string{"os.Remove", "os.RemoveAll", "os.Rename", "os.Truncate", ".Truncate("}
	for name, text := range nonTestSource(t) {
		for _, call := range forbidden {
			if strings.Contains(text, call) {
				t.Errorf("%s contains %q", name, call)
			}
		}
	}
}

// TestProgramMakesNoNetworkCalls checks REQ-SEC-4: no network packages are
// imported, and net/http is used only for content-type detection.
func TestProgramMakesNoNetworkCalls(t *testing.T) {
	banned := []string{`"net"`, `"net/rpc"`, `"net/smtp"`, `"net/http/httputil"`, `"crypto/tls"`, `"os/exec"`}
	httpUse := regexp.MustCompile(`\bhttp\.(\w+)`)
	for name, text := range nonTestSource(t) {
		for _, pkg := range banned {
			if strings.Contains(text, pkg) {
				t.Errorf("%s imports %s", name, pkg)
			}
		}
		for _, m := range httpUse.FindAllStringSubmatch(text, -1) {
			if m[1] != "DetectContentType" {
				t.Errorf("%s uses http.%s; net/http may only be used for DetectContentType", name, m[1])
			}
		}
	}
}

// TestRunKeepsOutputInsideOutDir checks REQ-SEC-3: names that try to climb out
// of the output directory are sanitized, and nothing is created outside it.
func TestRunKeepsOutputInsideOutDir(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	out := filepath.Join(dir, "out")
	writeExport(t, inPath, Export{
		Folders: []Folder{{ID: "f1", Name: "../../escape"}, {ID: "f2", Name: `..\..\win`}, {ID: "f3", Name: "/abs/folder"}},
		History: []Conversation{
			{ID: "c1", Name: "../../../evil", FolderID: "f1", Messages: []Message{{Role: "user", Files: []FileRef{{FileKey: "k", FileName: "../../x.txt"}}}}},
			{ID: "c2", Name: "/abs/path", FolderID: "f2", Messages: []Message{{Role: "user", Content: "x"}}},
			{ID: "c3", Name: `..\..\more`, FolderID: "f3", Messages: []Message{{Role: "user", Content: "x"}}},
		},
		Files: []FileBlob{{FileKey: "k", FileName: "../../x.txt", Content: "x"}},
	})

	for _, format := range []string{"both"} {
		if err := run([]string{"-in", inPath, "-out", out, "-format", format}, &bytes.Buffer{}); err != nil {
			t.Fatalf("run: %v", err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "export.json" && e.Name() != "out" {
			t.Errorf("created %q outside the output directory", e.Name())
		}
	}

	for rel := range snapshotTree(t, out) {
		for _, element := range strings.Split(rel, "/") {
			if element == ".." || element == "." || element == "" {
				t.Errorf("output path %q has the path element %q", rel, element)
			}
		}
	}
}

// TestGeneratedLinksAreRelative checks REQ-NAV-3: links between generated pages
// are relative paths, never absolute paths or URLs.
func TestGeneratedLinksAreRelative(t *testing.T) {
	outDir, _ := runWith(t, htmlSampleExport(), "-format", "html")
	reference := regexp.MustCompile(`(?:href|src)="([^"]*)"`)

	checked := 0
	for rel, page := range snapshotTree(t, outDir) {
		if !strings.HasSuffix(rel, ".html") {
			continue
		}
		for _, m := range reference.FindAllStringSubmatch(page, -1) {
			target := m[1]
			checked++
			if strings.HasPrefix(target, "/") || strings.Contains(target, "://") ||
				strings.HasPrefix(target, "file:") || strings.HasPrefix(target, "data:") {
				t.Errorf("%s has a non-relative link %q", rel, target)
			}
		}
	}
	if checked == 0 {
		t.Fatalf("found no links to check")
	}
}

// TestRenderMarkdownEdgeCases checks the Markdown layout rules in section 4.3,
// including empty messages, no messages, and the trailing newlines.
func TestRenderMarkdownEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		messages    []Message
		attachments []attachment
		want        string
	}{
		{"no messages", "", nil, nil, "# T\n\n## Messages\n\n"},
		{"id line", "x", nil, nil, "# T\n\n_id: `x`_\n\n## Messages\n\n"},
		{"one message", "", []Message{{Role: "user", Content: "hi"}}, nil, "# T\n\n## Messages\n\n### user\n\nhi\n\n"},
		{"content and role are trimmed", "", []Message{{Role: " user ", Content: "  hi  "}}, nil, "# T\n\n## Messages\n\n### user\n\nhi\n\n"},
		{"empty content", "", []Message{{Role: "user"}}, nil, "# T\n\n## Messages\n\n### user\n\n\n\n"},
		{"blank role", "", []Message{{Content: "hi"}}, nil, "# T\n\n## Messages\n\n### message\n\nhi\n\n"},
		{
			"attachment link ends with one newline", "",
			[]Message{{Role: "user", Content: "hi"}},
			[]attachment{{Name: "a", Rel: "assets/a b.txt"}},
			"# T\n\n## Messages\n\n### user\n\nhi\n\n## Attachments\n\n- **a**: [a b.txt](assets/a%20b.txt)\n",
		},
		{
			"attachment failure", "", nil,
			[]attachment{{Name: "b", Failure: "missing content for fileKey `k`"}},
			"# T\n\n## Messages\n\n## Attachments\n\n- **b**: _missing content for fileKey `k`_\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderMarkdown("T", tt.id, tt.messages, tt.attachments); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
