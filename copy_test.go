package main

import (
	"crypto/sha256"
	"encoding/base64"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	cspMeta   = regexp.MustCompile(`http-equiv="Content-Security-Policy" content="([^"]*)"`)
	scriptTag = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

// cspOf returns the page's Content-Security-Policy with HTML entities decoded,
// as a browser would read it.
func cspOf(t *testing.T, page string) string {
	t.Helper()
	m := cspMeta.FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("page has no Content-Security-Policy meta tag")
	}
	return html.UnescapeString(m[1])
}

// scriptOf returns the body of the page's only script element, or "" if it has none.
func scriptOf(t *testing.T, page string) string {
	t.Helper()
	all := scriptTag.FindAllStringSubmatch(page, -1)
	if len(all) > 1 {
		t.Fatalf("page has %d script elements, want at most 1", len(all))
	}
	if len(all) == 0 {
		return ""
	}
	return all[0][1]
}

func hashOf(script string) string {
	sum := sha256.Sum256([]byte(script))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

const pageWithCode = "Here is some code:\n\n```go\nfmt.Println(\"hi\")\n```\n"

func TestContentSecurityPolicy(t *testing.T) {
	if got := contentSecurityPolicy(""); got != baseCSP {
		t.Fatalf("policy without a script = %q, want the base policy", got)
	}

	got := contentSecurityPolicy("var a = 1;")
	want := baseCSP + "; script-src " + hashOf("var a = 1;")
	if got != want {
		t.Fatalf("policy with a script = %q, want %q", got, want)
	}
	for _, bad := range []string{"unsafe-eval", "script-src 'unsafe-inline'", "script-src *", "https:"} {
		if strings.Contains(got, bad) {
			t.Errorf("policy %q contains %q", got, bad)
		}
	}
}

func TestPageWithCodeBlockGetsCopyScriptAllowedByHash(t *testing.T) {
	r := newTestRenderer(t)
	out, err := r.renderConversation("T", "", conversationNav("F", "T", "../index.html"),
		[]Message{{Role: "assistant", Content: pageWithCode}}, nil)
	if err != nil {
		t.Fatalf("renderConversation: %v", err)
	}
	page := string(out)

	script := scriptOf(t, page)
	if script == "" {
		t.Fatalf("page with a code block has no script")
	}
	if script != copyScript {
		t.Errorf("served script differs from copyScript; the template altered it")
	}

	// The hash must match the exact bytes the browser will see.
	csp := cspOf(t, page)
	if want := baseCSP + "; script-src " + hashOf(script); csp != want {
		t.Errorf("policy = %q, want %q", csp, want)
	}
	if n := strings.Count(csp, "script-src"); n != 1 {
		t.Errorf("policy has %d script-src directives, want 1", n)
	}
}

func TestPagesWithoutCodeBlocksHaveNoScript(t *testing.T) {
	r := newTestRenderer(t)
	conv, err := r.renderConversation("T", "", conversationNav("F", "T", "../index.html"),
		[]Message{{Role: "user", Content: "just `inline code` and prose"}}, nil)
	if err != nil {
		t.Fatalf("renderConversation: %v", err)
	}
	folder, err := r.renderFolder("F", "../index.html", []indexEntry{{Title: "T", File: "T.html"}})
	if err != nil {
		t.Fatalf("renderFolder: %v", err)
	}
	index, err := r.renderIndex([]indexEntry{{Folder: "F", Title: "T", Href: "F/T.html"}})
	if err != nil {
		t.Fatalf("renderIndex: %v", err)
	}

	for name, page := range map[string]string{
		"conversation without code": string(conv),
		"folder page":               string(folder),
		"main index":                string(index),
	} {
		if strings.Contains(page, "<script") {
			t.Errorf("%s contains a script element", name)
		}
		if got := cspOf(t, page); got != baseCSP {
			t.Errorf("%s policy = %q, want the base policy", name, got)
		}
	}
}

func TestChatContentCannotAddScripts(t *testing.T) {
	r := newTestRenderer(t)
	content := "<script>alert(1)</script>\n\n" + pageWithCode + "\n\ntext <script src=\"https://evil.example/x.js\"></script>\n\n```html\n<script>alert(2)</script>\n```\n"
	out, err := r.renderConversation("T", "", conversationNav("F", "T", "../index.html"),
		[]Message{{Role: "user", Content: content}}, nil)
	if err != nil {
		t.Fatalf("renderConversation: %v", err)
	}
	page := string(out)

	if n := strings.Count(page, "<script"); n != 1 {
		t.Fatalf("page has %d script elements, want only the copy-button script", n)
	}
	if strings.Contains(page, "<script src") {
		t.Errorf("an external script element reached the page")
	}
	if !strings.Contains(page, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("the chat's script text should be visible, escaped")
	}
}

func TestCopyScriptHasNoPitfallsForTheTemplate(t *testing.T) {
	for _, bad := range []string{"</", "<!--", "-->", "/*", "//"} {
		if strings.Contains(copyScript, bad) {
			t.Errorf("copyScript contains %q, which html/template or the parser could treat specially", bad)
		}
	}
	for _, want := range []string{
		`querySelectorAll("pre > code")`,
		"navigator.clipboard.writeText",
		`document.execCommand("copy")`,
		"textContent",
		`"aria-label"`,
		`"role", "status"`,
	} {
		if !strings.Contains(copyScript, want) {
			t.Errorf("copyScript does not contain %q", want)
		}
	}
}

func TestRunWritesCopyScriptOnlyOnPagesWithCode(t *testing.T) {
	outDir, _ := runWith(t, htmlSampleExport(), "-format", "html")

	withCode := filepath.Join(outDir, "Project A", "First_ Chat.html")
	if script := scriptOf(t, readFile(t, withCode)); script != copyScript {
		t.Errorf("%s should carry the copy script", withCode)
	}

	for _, p := range []string{
		filepath.Join(outDir, "Project A", "First_ Chat_2.html"),
		filepath.Join(outDir, "Unfiled", shortHashFromString("c2", 10)+"_2.html"),
		filepath.Join(outDir, "Project A", "index.html"),
		filepath.Join(outDir, "Unfiled", "index.html"),
		filepath.Join(outDir, "index.html"),
	} {
		page := readFile(t, p)
		if strings.Contains(page, "<script") {
			t.Errorf("%s has a script element but no code block", p)
		}
		if got := cspOf(t, page); got != baseCSP {
			t.Errorf("%s policy = %q, want the base policy", p, got)
		}
	}

	if _, err := os.Stat(withCode); err != nil {
		t.Fatalf("stat: %v", err)
	}
}
