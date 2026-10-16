package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newTestRenderer(t *testing.T) *htmlRenderer {
	t.Helper()
	r, err := newHTMLRenderer()
	if err != nil {
		t.Fatalf("newHTMLRenderer: %v", err)
	}
	return r
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		in      string
		want    outputFormat
		wantErr bool
	}{
		{"md", formatMarkdown, false},
		{"html", formatHTML, false},
		{"both", formatBoth, false},
		{" HTML ", formatHTML, false},
		{"Both", formatBoth, false},
		{"", "", true},
		{"pdf", "", true},
		{"md,html", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseFormat(tt.in)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidFormat) {
					t.Fatalf("err = %v, want ErrInvalidFormat", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("parseFormat(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
			}
		})
	}
}

func TestOutputFormatWrites(t *testing.T) {
	tests := []struct {
		format   outputFormat
		markdown bool
		html     bool
	}{
		{formatMarkdown, true, false},
		{formatHTML, false, true},
		{formatBoth, true, true},
	}
	for _, tt := range tests {
		if tt.format.writesMarkdown() != tt.markdown || tt.format.writesHTML() != tt.html {
			t.Errorf("%s: writesMarkdown=%v writesHTML=%v", tt.format, tt.format.writesMarkdown(), tt.format.writesHTML())
		}
	}
}

func TestMarkdownToHTMLRendersFormatting(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"heading", "# Title", "<h1>Title</h1>"},
		{"single newline is a line break", "Linux: a\nWindows: b", "<p>Linux: a<br>\nWindows: b</p>"},
		{"blank line still starts a paragraph", "one\n\ntwo", "<p>one</p>\n<p>two</p>"},
		{"code block lines are not broken", "```\na\nb\n```", "<pre><code>a\nb\n</code></pre>"},
		{"list items are not broken", "- a\n- b", "<li>a</li>\n<li>b</li>"},
		{"emphasis", "**bold** and *italic*", "<strong>bold</strong> and <em>italic</em>"},
		{"inline code", "use `go test`", "<code>go test</code>"},
		{"fenced code with language", "```go\nfmt.Println(1)\n```", `<pre><code class="language-go">fmt.Println(1)`},
		{"code is escaped", "```\na < b && c > d\n```", "a &lt; b &amp;&amp; c &gt; d"},
		{"list", "- one\n- two", "<li>one</li>"},
		{"table", "| a | b |\n|---|---|\n| 1 | 2 |", "<th>a</th>"},
		{"strikethrough", "~~gone~~", "<del>gone</del>"},
		{"task list", "- [x] done", `type="checkbox"`},
		{"safe link", "[site](https://example.com/a)", `href="https://example.com/a"`},
	}

	r := newTestRenderer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.markdownToHTML(tt.in)
			if err != nil {
				t.Fatalf("markdownToHTML: %v", err)
			}
			if !strings.Contains(string(got), tt.want) {
				t.Fatalf("output %q does not contain %q", got, tt.want)
			}
		})
	}
}

func TestMarkdownToHTMLEscapesRawHTML(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantText  string
		forbidden []string
	}{
		{"script block", "<script>alert(1)</script>", "&lt;script&gt;alert(1)&lt;/script&gt;", []string{"<script"}},
		{"div block", "<div>hi</div>", "&lt;div&gt;hi&lt;/div&gt;", []string{"<div"}},
		{"inline image with handler", "text <img src=x onerror=alert(1)> more", "&lt;img src=x onerror=alert(1)&gt;", []string{"<img"}},
		{"inline tag in prose", "use the <b>bold</b> tag", "&lt;b&gt;bold&lt;/b&gt;", []string{"<b>"}},
		{"html comment", "<!-- note -->\n\ntext", "&lt;!-- note --&gt;", []string{"<!--"}},
	}

	r := newTestRenderer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.markdownToHTML(tt.in)
			if err != nil {
				t.Fatalf("markdownToHTML: %v", err)
			}
			out := string(got)
			if !strings.Contains(out, tt.wantText) {
				t.Errorf("output %q does not contain the escaped text %q", out, tt.wantText)
			}
			for _, bad := range append(tt.forbidden, "raw HTML omitted") {
				if strings.Contains(out, bad) {
					t.Errorf("output %q contains %q", out, bad)
				}
			}
		})
	}
}

func TestMarkdownToHTMLNeutralizesDangerousURLs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		bad  string
	}{
		{"javascript link", "[x](javascript:alert(1))", `href="javascript`},
		{"mixed case javascript link", "[x](JaVaScRiPt:alert(1))", `href="JaVaScRiPt`},
		{"vbscript link", "[x](vbscript:msgbox(1))", `href="vbscript`},
		{"file link", "[x](file:///etc/passwd)", `href="file:`},
		{"data html link", "[x](data:text/html;base64,PHNjcmlwdD4=)", `href="data:text/html`},
		{"data html image", "![x](data:text/html;base64,PHNjcmlwdD4=)", `src="data:text/html`},
	}

	r := newTestRenderer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.markdownToHTML(tt.in)
			if err != nil {
				t.Fatalf("markdownToHTML: %v", err)
			}
			if strings.Contains(string(got), tt.bad) {
				t.Fatalf("output %q contains %q", got, tt.bad)
			}
		})
	}
}

func TestRoleClass(t *testing.T) {
	tests := map[string]string{
		"user":                 "role-user",
		"USER":                 "role-user",
		"assistant":            "role-assistant",
		"system":               "role-other",
		`"><script>x</script>`: "role-other",
	}
	for role, want := range tests {
		if got := roleClass(role); got != want {
			t.Errorf("roleClass(%q) = %q, want %q", role, got, want)
		}
	}
}

func TestIsImageName(t *testing.T) {
	tests := map[string]bool{
		"assets/pic.png": true,
		"a.JPG":          true,
		"a.jpeg":         true,
		"a.svg":          true,
		"a.txt":          false,
		"png":            false,
		"a.png.txt":      false,
	}
	for name, want := range tests {
		if got := isImageName(name); got != want {
			t.Errorf("isImageName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestRenderConversation(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.renderConversation(
		"<b>Title</b> & more",
		"c1",
		conversationNav("Folder <x>", "<b>Title</b> & more", "../index.html"),
		[]Message{
			{Role: "user", Content: " hello **there** "},
			{Role: `"><script>alert(1)</script>`, Content: "x"},
			{Role: "", Content: "no role"},
		},
		[]attachment{
			{Name: "pic", Rel: "assets/pic.png"},
			{Name: "notes", Rel: "assets/my notes (1).txt"},
			{Name: "lost <1>", Failure: "missing content for fileKey `k<1>`"},
		},
	)
	if err != nil {
		t.Fatalf("renderConversation: %v", err)
	}
	page := string(out)

	for _, want := range []string{
		"<!DOCTYPE html>",
		`<meta http-equiv="Content-Security-Policy" content="default-src 'none'`,
		"<title>&lt;b&gt;Title&lt;/b&gt; &amp; more</title>",
		"<h1>&lt;b&gt;Title&lt;/b&gt; &amp; more</h1>",
		"<code>c1</code>",
		`<section class="message role-user">`,
		"<strong>there</strong>",
		`<section class="message role-other">`,
		"<h3>message</h3>",
		`<img src="assets/pic.png" alt="pic" loading="lazy">`,
		`<a href="assets/my%20notes%20%281%29.txt">my notes (1).txt</a>`,
		"<strong>lost &lt;1&gt;</strong>",
		"missing content for fileKey `k&lt;1&gt;`",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page does not contain %q", want)
		}
	}

	for _, bad := range []string{"<b>Title", "<script", `alt="notes"`, "onerror"} {
		if strings.Contains(page, bad) {
			t.Errorf("page contains %q", bad)
		}
	}
	if strings.Contains(page, `<img src="assets/my`) {
		t.Errorf("a text attachment was rendered as an image")
	}
}

func TestRenderConversationOmitsEmptySections(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.renderConversation("T", "", conversationNav("F", "T", "../index.html"), nil, nil)
	if err != nil {
		t.Fatalf("renderConversation: %v", err)
	}
	page := string(out)
	for _, bad := range []string{"Attachments", `class="meta"`} {
		if strings.Contains(page, bad) {
			t.Errorf("page contains %q", bad)
		}
	}
}

func TestRenderIndexGroupsAndSorts(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.renderIndex([]indexEntry{
		{Folder: "zeta", Title: "Z one", Href: "zeta/Z%20one.html"},
		{Folder: unfiledName, Title: "Loose", Href: "Unfiled/Loose.html"},
		{Folder: "Alpha", Title: "A <one>", Href: "Alpha/A.html"},
		{Folder: "alpha2", Title: "A two", Href: "alpha2/B.html"},
		{Folder: "Alpha", Title: "A two", Href: "Alpha/B.html"},
	})
	if err != nil {
		t.Fatalf("renderIndex: %v", err)
	}
	page := string(out)

	order := []string{
		`<h2><a href="Alpha/index.html">Alpha</a></h2>`,
		`<h2><a href="alpha2/index.html">alpha2</a></h2>`,
		`<h2><a href="zeta/index.html">zeta</a></h2>`,
		`<h2><a href="Unfiled/index.html">Unfiled</a></h2>`,
	}
	last := -1
	for _, want := range order {
		i := strings.Index(page, want)
		if i < 0 {
			t.Fatalf("index does not contain %q", want)
		}
		if i < last {
			t.Fatalf("%q is out of order", want)
		}
		last = i
	}

	for _, want := range []string{
		`<a href="zeta/Z%20one.html">Z one</a>`,
		`<a href="Alpha/A.html">A &lt;one&gt;</a>`,
		"<title>" + indexTitle + "</title>",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("index does not contain %q", want)
		}
	}
	if strings.Contains(page, "<one>") {
		t.Errorf("index contains an unescaped title")
	}
}

func htmlSampleExport() Export {
	return Export{
		Version: 1,
		Folders: []Folder{{ID: "f1", Name: "Project A", Type: "chat"}},
		History: []Conversation{
			{
				ID:       "c1",
				Name:     "First: Chat",
				FolderID: "f1",
				Messages: []Message{
					{Role: "user", Content: "<script>alert(1)</script>\n\nhello **bold**"},
					{Role: "assistant", Content: "```go\nfmt.Println(1)\n```", Files: []FileRef{{FileKey: "k1", FileName: "my notes (1).txt"}}},
				},
			},
			{ID: "c2", FolderID: nil, Messages: []Message{{Role: "user", Content: "loose chat"}}},
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
	}
}

func runWith(t *testing.T, export Export, extraArgs ...string) (outDir, summary string) {
	t.Helper()
	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	outDir = filepath.Join(dir, "out")
	writeExport(t, inPath, export)

	var stdout bytes.Buffer
	args := append([]string{"-in", inPath, "-out", outDir}, extraArgs...)
	if err := run(args, &stdout); err != nil {
		t.Fatalf("run %v failed: %v", extraArgs, err)
	}
	return outDir, stdout.String()
}

func TestRunHTMLFormat(t *testing.T) {
	outDir, summary := runWith(t, htmlSampleExport(), "-format", "html")

	if !strings.Contains(summary, "Wrote 3 of 3 HTML files") || !strings.Contains(summary, "index.html written") {
		t.Fatalf("unexpected summary: %q", summary)
	}
	if strings.Contains(summary, "markdown") {
		t.Fatalf("summary mentions markdown for an html-only run: %q", summary)
	}

	var mdFiles []string
	err := filepath.WalkDir(outDir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
			mdFiles = append(mdFiles, p)
		}
		return err
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(mdFiles) != 0 {
		t.Fatalf("html-only run wrote Markdown files: %v", mdFiles)
	}

	first := readFile(t, filepath.Join(outDir, "Project A", "First_ Chat.html"))
	for _, want := range []string{
		"<strong>bold</strong>",
		"&lt;script&gt;alert(1)&lt;/script&gt;",
		`<pre><code class="language-go">fmt.Println(1)`,
		`<a href="assets/my%20notes%20%281%29.txt">my notes (1).txt</a>`,
	} {
		if !strings.Contains(first, want) {
			t.Errorf("first conversation missing %q", want)
		}
	}
	if strings.Contains(first, "<script") {
		t.Errorf("first conversation contains an unescaped script tag")
	}

	second := readFile(t, filepath.Join(outDir, "Project A", "First_ Chat_2.html"))
	if !strings.Contains(second, `<img src="assets/pic.png"`) {
		t.Errorf("duplicate-name conversation missing inline image")
	}
	if got := readFile(t, filepath.Join(outDir, "Project A", "assets", "my notes (1).txt")); got != "note body" {
		t.Errorf("attachment content = %q", got)
	}

	unfiled := readFile(t, filepath.Join(outDir, "Unfiled", shortHashFromString("c2", 10)+"_2.html"))
	if !strings.Contains(unfiled, "<h1>Conversation 2</h1>") {
		t.Errorf("unfiled conversation missing its fallback title")
	}

	index := readFile(t, filepath.Join(outDir, "index.html"))
	for _, want := range []string{
		`<a href="Project%20A/First_%20Chat.html">First: Chat</a>`,
		`<a href="Project%20A/First_%20Chat_2.html">First: Chat</a>`,
		fmtIndexLink(shortHashFromString("c2", 10)+"_2.html", "Conversation 2"),
	} {
		if !strings.Contains(index, want) {
			t.Errorf("index missing %q in:\n%s", want, index)
		}
	}
	if strings.Index(index, ">Project A</a></h2>") > strings.Index(index, ">Unfiled</a></h2>") {
		t.Errorf("Unfiled should be listed after named folders")
	}
}

func fmtIndexLink(file, title string) string {
	return `<a href="Unfiled/` + file + `">` + title + `</a>`
}

func TestRunBothFormats(t *testing.T) {
	outDir, summary := runWith(t, htmlSampleExport(), "-format", "both")

	for _, want := range []string{"Wrote 3 of 3 markdown files", "Wrote 3 of 3 HTML files"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary %q does not contain %q", summary, want)
		}
	}
	if !strings.HasPrefix(summary, "Done. ") {
		t.Errorf("summary should start with Done.: %q", summary)
	}

	for _, name := range []string{"First_ Chat.md", "First_ Chat.html", "First_ Chat_2.md", "First_ Chat_2.html"} {
		if _, err := os.Stat(filepath.Join(outDir, "Project A", name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outDir, "index.html")); err != nil {
		t.Errorf("missing index.html: %v", err)
	}

	md := readFile(t, filepath.Join(outDir, "Project A", "First_ Chat.md"))
	if !strings.Contains(md, "<script>alert(1)</script>") {
		t.Errorf("markdown output should keep the original text unchanged")
	}
}

func TestRunMarkdownOnlyWritesNoHTML(t *testing.T) {
	outDir, summary := runWith(t, htmlSampleExport())

	if strings.Contains(summary, "HTML files") || strings.Contains(summary, "index.html") {
		t.Fatalf("default run reports HTML output: %q", summary)
	}
	if _, err := os.Stat(filepath.Join(outDir, "index.html")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("default run wrote index.html: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "Project A", "First_ Chat.html")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("default run wrote an HTML file: %v", err)
	}
}

func TestRunHTMLOverwriteBehavior(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	outDir := filepath.Join(dir, "out")
	writeExport(t, inPath, htmlSampleExport())

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"first run writes everything", []string{"-format", "html"}, []string{"Wrote 3 of 3 HTML", "index.html written"}},
		{"rerun keeps existing files", []string{"-format", "html"}, []string{"Wrote 0 of 3 HTML", "index.html kept"}},
		{"overwrite rewrites them", []string{"-format", "html", "-overwrite"}, []string{"Wrote 3 of 3 HTML", "index.html written"}},
	}
	for _, tt := range tests {
		var stdout bytes.Buffer
		args := append([]string{"-in", inPath, "-out", outDir}, tt.args...)
		if err := run(args, &stdout); err != nil {
			t.Fatalf("%s: run failed: %v", tt.name, err)
		}
		for _, want := range tt.want {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("%s: summary %q does not contain %q", tt.name, stdout.String(), want)
			}
		}
	}
}

func TestRunRejectsInvalidFormat(t *testing.T) {
	err := run([]string{"-in", "unused.json", "-format", "pdf"}, &bytes.Buffer{})
	if !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("err = %v, want ErrInvalidFormat", err)
	}
}

func TestRunHTMLOutputIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}

	outDir, _ := runWith(t, htmlSampleExport(), "-format", "html")
	for _, p := range []string{
		filepath.Join(outDir, "index.html"),
		filepath.Join(outDir, "Project A", "First_ Chat.html"),
	} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %q: %v", p, err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("%s mode = %o, want 600", p, got)
		}
	}
}
