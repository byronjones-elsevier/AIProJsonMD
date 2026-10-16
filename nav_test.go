package main

import (
	"bytes"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestConversationNav(t *testing.T) {
	got := conversationNav("Project A", "First chat", "../index.html")
	want := navData{Crumbs: []crumb{
		{Label: "Main index", Href: "../index.html"},
		{Label: "Project A", Href: "index.html"},
		{Label: "First chat"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("conversationNav = %+v, want %+v", got, want)
	}
}

func TestFolderNav(t *testing.T) {
	got := folderNav("Project A", "../index.html")
	want := navData{Crumbs: []crumb{{Label: "Main index", Href: "../index.html"}, {Label: "Project A"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("folderNav = %+v, want %+v", got, want)
	}
}

func TestRenderConversationShowsBreadcrumbAndLinks(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.renderConversation("<b>Title</b> & more", "", conversationNav("Folder <x>", "<b>Title</b> & more", "../index.html"), nil, nil)
	if err != nil {
		t.Fatalf("renderConversation: %v", err)
	}
	page := string(out)

	for _, want := range []string{
		`<nav class="breadcrumb" aria-label="Breadcrumb">`,
		`<li><a href="../index.html">Main index</a></li>`,
		`<li><a href="index.html">Folder &lt;x&gt;</a></li>`,
		`<li aria-current="page">&lt;b&gt;Title&lt;/b&gt; &amp; more</li>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page does not contain %q", want)
		}
	}
	// The breadcrumb is the only navigation; it must not be repeated below.
	for _, link := range []string{`href="../index.html"`, `href="index.html"`} {
		if n := strings.Count(page, link); n != 1 {
			t.Errorf("%s appears %d times, want once (in the breadcrumb)", link, n)
		}
	}
	if strings.Contains(page, "Up to") || strings.Contains(page, "&uarr;") {
		t.Errorf("page repeats the breadcrumb links in a second navigation line")
	}
	if strings.Contains(page, `<a href="`+"index.html"+`">&lt;b&gt;Title`) {
		t.Errorf("the current page should not be a link in the breadcrumb")
	}
}

func TestRenderFolder(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.renderFolder("My <Folder>", "../index.html", []indexEntry{
		{Title: "A <one>", File: "A one (1).html"},
		{Title: "Two", File: "Two.html"},
	})
	if err != nil {
		t.Fatalf("renderFolder: %v", err)
	}
	page := string(out)

	for _, want := range []string{
		"<title>My &lt;Folder&gt;</title>",
		"<h1>My &lt;Folder&gt;</h1>",
		`<li><a href="../index.html">Main index</a></li>`,
		`<li aria-current="page">My &lt;Folder&gt;</li>`,
		`<li><a href="A%20one%20%281%29.html">A &lt;one&gt;</a></li>`,
		`<li><a href="Two.html">Two</a></li>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("folder page does not contain %q", want)
		}
	}
	if n := strings.Count(page, `href="../index.html"`); n != 1 {
		t.Errorf("the main index link appears %d times, want once (in the breadcrumb)", n)
	}
	if strings.Contains(page, "Up to") || strings.Contains(page, "&uarr;") {
		t.Errorf("folder page repeats the breadcrumb links in a second navigation line")
	}
	if strings.Contains(page, "<Folder>") || strings.Contains(page, "<one>") {
		t.Errorf("folder page contains an unescaped name")
	}
}

var linkAttr = regexp.MustCompile(`(?:href|src)="([^"]*)"`)

// localLinkTargets returns, for every HTML page under root, the files its local
// links point at, and fails the test for any link whose target does not exist.
func localLinkTargets(t *testing.T, root string) map[string][]string {
	t.Helper()
	targets := make(map[string][]string)

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		for _, m := range linkAttr.FindAllStringSubmatch(readFile(t, p), -1) {
			u, perr := url.Parse(html.UnescapeString(m[1]))
			if perr != nil {
				t.Errorf("%s: unparseable link %q: %v", p, m[1], perr)
				continue
			}
			if u.Scheme != "" || u.Host != "" {
				continue
			}
			target := filepath.Join(filepath.Dir(p), filepath.FromSlash(u.Path))
			if _, serr := os.Stat(target); serr != nil {
				t.Errorf("%s: link %q does not resolve: %v", p, m[1], serr)
				continue
			}
			targets[p] = append(targets[p], target)
		}
		if _, ok := targets[p]; !ok {
			targets[p] = nil
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return targets
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestRunHTMLNavigation(t *testing.T) {
	outDir, summary := runWith(t, htmlSampleExport(), "-format", "html")

	if !strings.Contains(summary, "index.html written, 2 of 2 folder index pages written") {
		t.Fatalf("unexpected summary: %q", summary)
	}

	rootIndex := filepath.Join(outDir, "index.html")
	projectDir := filepath.Join(outDir, "Project A")
	unfiledDir := filepath.Join(outDir, "Unfiled")
	targets := localLinkTargets(t, outDir)

	conversations := []string{
		filepath.Join(projectDir, "First_ Chat.html"),
		filepath.Join(projectDir, "First_ Chat_2.html"),
		filepath.Join(unfiledDir, shortHashFromString("c2", 10)+"_2.html"),
	}
	if len(targets) != 6 {
		t.Fatalf("found %d HTML pages, want 6 (main index, 2 folder pages, 3 conversations)", len(targets))
	}

	// The main index links down to each folder page.
	for _, dir := range []string{projectDir, unfiledDir} {
		if !contains(targets[rootIndex], filepath.Join(dir, "index.html")) {
			t.Errorf("main index does not link to %s", filepath.Join(dir, "index.html"))
		}
	}

	// Each folder page links up to the main index and down to its conversations.
	for _, dir := range []string{projectDir, unfiledDir} {
		page := filepath.Join(dir, "index.html")
		if !contains(targets[page], rootIndex) {
			t.Errorf("%s does not link to the main index", page)
		}
		if !strings.Contains(readFile(t, page), `aria-current="page"`) {
			t.Errorf("%s has no current-page marker in the breadcrumb", page)
		}
	}
	for _, conv := range conversations {
		folderPage := filepath.Join(filepath.Dir(conv), "index.html")
		if !contains(targets[folderPage], conv) {
			t.Errorf("%s does not list %s", folderPage, conv)
		}
	}

	// Each conversation page has a breadcrumb, a link up to its folder, and a
	// link back to the main index.
	for _, conv := range conversations {
		page := readFile(t, conv)
		if !strings.Contains(page, `<nav class="breadcrumb"`) || !strings.Contains(page, `aria-current="page"`) {
			t.Errorf("%s has no breadcrumb", conv)
		}
		if !contains(targets[conv], filepath.Join(filepath.Dir(conv), "index.html")) {
			t.Errorf("%s does not link up to its folder page", conv)
		}
		if !contains(targets[conv], rootIndex) {
			t.Errorf("%s does not link to the main index", conv)
		}
	}

	if !strings.Contains(readFile(t, conversations[0]), `<a href="../index.html">Main index</a>`) {
		t.Errorf("conversation page should reach the main index with ../index.html")
	}
	if strings.Contains(readFile(t, rootIndex), `class="breadcrumb"`) {
		t.Errorf("the main index should not have a breadcrumb")
	}
}

func TestRunReservesIndexNames(t *testing.T) {
	export := Export{
		Folders: []Folder{
			{ID: "f1", Name: "Docs"},
			{ID: "f2", Name: "docs"},
			{ID: "f3", Name: "index.html"},
		},
		History: []Conversation{
			{ID: "c1", Name: "index", FolderID: "f1", Messages: []Message{{Role: "user", Content: "first"}}},
			{ID: "c2", Name: "Index", FolderID: "f1", Messages: []Message{{Role: "user", Content: "second"}}},
			{ID: "c3", Name: "x", FolderID: "f2", Messages: []Message{{Role: "user", Content: "third"}}},
			{ID: "c4", Name: "y", FolderID: "f3", Messages: []Message{{Role: "user", Content: "fourth"}}},
		},
	}
	outDir, _ := runWith(t, export, "-format", "both")

	info, err := os.Stat(filepath.Join(outDir, "index.html"))
	if err != nil || info.IsDir() {
		t.Fatalf("the main index.html must be a file: %v", err)
	}

	for _, name := range []string{"index_2.md", "index_2.html", "Index_3.md", "Index_3.html"} {
		if _, err := os.Stat(filepath.Join(outDir, "Docs", name)); err != nil {
			t.Errorf("missing Docs/%s: %v", name, err)
		}
	}
	folderPage := readFile(t, filepath.Join(outDir, "Docs", "index.html"))
	for _, want := range []string{`<a href="index_2.html">index</a>`, `<a href="Index_3.html">Index</a>`, "<h1>Docs</h1>"} {
		if !strings.Contains(folderPage, want) {
			t.Errorf("Docs folder page does not contain %q", want)
		}
	}
	if strings.Contains(folderPage, "first") {
		t.Errorf("the Docs folder page was replaced by a conversation")
	}

	for _, dir := range []string{"docs_2", "index.html_2"} {
		if _, err := os.Stat(filepath.Join(outDir, dir, "index.html")); err != nil {
			t.Errorf("missing folder page in %s: %v", dir, err)
		}
	}

	localLinkTargets(t, outDir)
}

func TestRunHTMLFolderPagesFollowOverwriteRules(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "export.json")
	outDir := filepath.Join(dir, "out")
	writeExport(t, inPath, htmlSampleExport())

	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"-format", "html"}, "index.html written, 2 of 2 folder index pages written"},
		{[]string{"-format", "html"}, "index.html kept, 0 of 2 folder index pages written"},
		{[]string{"-format", "html", "-overwrite"}, "index.html written, 2 of 2 folder index pages written"},
	} {
		var stdout bytes.Buffer
		if err := run(append([]string{"-in", inPath, "-out", outDir}, tt.args...), &stdout); err != nil {
			t.Fatalf("run %v: %v", tt.args, err)
		}
		if !strings.Contains(stdout.String(), tt.want) {
			t.Fatalf("summary %q does not contain %q", stdout.String(), tt.want)
		}
	}
}

func TestMarkdownOnlyRunWritesNoFolderPages(t *testing.T) {
	outDir, _ := runWith(t, htmlSampleExport())
	for _, p := range []string{
		filepath.Join(outDir, "Project A", "index.html"),
		filepath.Join(outDir, "Unfiled", "index.html"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("markdown-only run wrote %s: %v", p, err)
		}
	}
}
