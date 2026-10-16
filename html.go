package main

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// indexTitle is the heading and page title of the generated index.html.
const indexTitle = "AI Pro Export"

// pageStyle is shared by the conversation and index pages. It is static, has no
// external resources, and follows the system light or dark preference.
const pageStyle = `:root{color-scheme:light dark;--bg:#fff;--fg:#1f2328;--muted:#59636e;--border:#d1d9e0;--user:#f6f8fa;--code:#f6f8fa}
@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--fg:#e6edf3;--muted:#9198a1;--border:#3d444d;--user:#161b22;--code:#161b22}}
body{margin:0;background:var(--bg);color:var(--fg);font:16px/1.6 system-ui,-apple-system,"Segoe UI",sans-serif}
main{max-width:52rem;margin:0 auto;padding:2rem 1rem}
.meta{color:var(--muted)}
.message{border:1px solid var(--border);border-radius:8px;padding:.5rem 1rem;margin:1rem 0}
.message h3{margin:.25rem 0;font-size:.85rem;text-transform:uppercase;letter-spacing:.05em;color:var(--muted)}
.role-user{background:var(--user)}
.body{overflow-wrap:anywhere}
pre{background:var(--code);border:1px solid var(--border);border-radius:6px;padding:.75rem;overflow:auto}
code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:.9em}
:not(pre)>code{background:var(--code);padding:.1em .3em;border-radius:4px}
table{border-collapse:collapse;display:block;overflow:auto}
th,td{border:1px solid var(--border);padding:.3rem .6rem}
blockquote{margin:0;padding-left:1rem;border-left:4px solid var(--border);color:var(--muted)}
img{max-width:100%}`

// pageHead is the shared document head. The Content-Security-Policy blocks
// scripts and every external resource, so a hostile chat cannot run code or
// phone home even if an escaping bug were introduced.
const pageHead = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; base-uri 'none'">
<title>{{.Title}}</title>
<style>` + pageStyle + `</style>
</head>
<body>
<main>
`

const pageTemplate = pageHead + `<h1>{{.Title}}</h1>
{{if .ID}}<p class="meta">id: <code>{{.ID}}</code></p>
{{end}}<h2>Messages</h2>
{{range .Messages}}<section class="message {{.RoleClass}}">
<h3>{{.Role}}</h3>
<div class="body">
{{.Body}}</div>
</section>
{{end}}{{if .Attachments}}<h2>Attachments</h2>
<ul class="attachments">
{{range .Attachments}}<li><strong>{{.Name}}</strong>: {{if .Failure}}<em>{{.Failure}}</em>{{else}}<a href="{{.Href}}">{{if .IsImage}}<img src="{{.Href}}" alt="{{.Name}}" loading="lazy"><br>{{end}}{{.LinkText}}</a>{{end}}</li>
{{end}}</ul>
{{end}}</main>
</body>
</html>
`

const indexTemplate = pageHead + `<h1>{{.Title}}</h1>
{{range .Folders}}<h2>{{.Name}}</h2>
<ul>
{{range .Conversations}}<li><a href="{{.Href}}">{{.Title}}</a></li>
{{end}}</ul>
{{end}}</main>
</body>
</html>
`

// pageData is the model for one conversation page.
type pageData struct {
	Title       string
	ID          string
	Messages    []messageData
	Attachments []attachmentData
}

// messageData is one rendered chat turn. Body is HTML produced by the Markdown
// renderer, which never emits raw HTML from the chat.
type messageData struct {
	Role      string
	RoleClass string
	Body      template.HTML
}

// attachmentData is one attachment list item.
type attachmentData struct {
	Name     string
	Href     string
	LinkText string
	IsImage  bool
	Failure  string
}

// indexEntry is one conversation listed on the index page.
type indexEntry struct {
	Folder string
	Title  string
	Href   string
}

// indexFolder groups index entries under a folder name.
type indexFolder struct {
	Name          string
	Conversations []indexEntry
}

// indexData is the model for index.html.
type indexData struct {
	Title   string
	Folders []indexFolder
}

// htmlRenderer renders conversations and the index page as HTML.
type htmlRenderer struct {
	markdown goldmark.Markdown
	page     *template.Template
	index    *template.Template
}

// newHTMLRenderer parses the page templates and builds a Markdown renderer
// that escapes, rather than passes through or drops, raw HTML found in chats.
func newHTMLRenderer() (*htmlRenderer, error) {
	page, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse page template: %w", err)
	}
	index, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse index template: %w", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.TaskList),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(escapeRawHTML{}, 100))),
	)
	return &htmlRenderer{markdown: md, page: page, index: index}, nil
}

// escapeRawHTML renders raw HTML nodes as visible, escaped text. goldmark's
// safe mode would otherwise replace them with a comment and lose the text,
// and unsafe mode would let chat content inject markup.
type escapeRawHTML struct{}

// RegisterFuncs overrides the default raw HTML node renderers.
func (escapeRawHTML) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindRawHTML, renderRawHTML)
	reg.Register(ast.KindHTMLBlock, renderHTMLBlock)
}

func renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.RawHTML)
	for i := range n.Segments.Len() {
		segment := n.Segments.At(i)
		_, _ = w.Write(util.EscapeHTML(segment.Value(source)))
	}
	return ast.WalkSkipChildren, nil
}

func renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.HTMLBlock)
	if entering {
		_, _ = w.WriteString(`<pre class="raw-html">`)
		for i := range n.Lines().Len() {
			line := n.Lines().At(i)
			_, _ = w.Write(util.EscapeHTML(line.Value(source)))
		}
		return ast.WalkContinue, nil
	}
	if n.HasClosure() {
		_, _ = w.Write(util.EscapeHTML(n.ClosureLine.Value(source)))
	}
	_, _ = w.WriteString("</pre>\n")
	return ast.WalkContinue, nil
}

// markdownToHTML converts one message body to HTML.
func (h *htmlRenderer) markdownToHTML(content string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := h.markdown.Convert([]byte(content), &buf); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return template.HTML(buf.String()), nil //nolint:gosec // goldmark output with raw HTML escaped by escapeRawHTML
}

// roleClass maps a chat role to a fixed CSS class, so arbitrary role text from
// the export never reaches an attribute.
func roleClass(role string) string {
	switch strings.ToLower(role) {
	case "user":
		return "role-user"
	case "assistant":
		return "role-assistant"
	default:
		return "role-other"
	}
}

// renderConversation renders one conversation page.
func (h *htmlRenderer) renderConversation(title, id string, messages []Message, attachments []attachment) ([]byte, error) {
	data := pageData{Title: title, ID: id}

	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role == "" {
			role = "message"
		}
		body, err := h.markdownToHTML(strings.TrimSpace(m.Content))
		if err != nil {
			return nil, err
		}
		data.Messages = append(data.Messages, messageData{Role: role, RoleClass: roleClass(role), Body: body})
	}

	for _, a := range attachments {
		item := attachmentData{Name: a.Name, Failure: a.Failure}
		if a.Failure == "" {
			item.Href = escapeLinkPath(a.Rel)
			item.LinkText = filepath.Base(a.Rel)
			item.IsImage = isImageName(a.Rel)
		}
		data.Attachments = append(data.Attachments, item)
	}

	var buf bytes.Buffer
	if err := h.page.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render conversation page: %w", err)
	}
	return buf.Bytes(), nil
}

// renderIndex renders index.html, grouping entries by folder with the Unfiled
// group last and the rest in alphabetical order.
func (h *htmlRenderer) renderIndex(entries []indexEntry) ([]byte, error) {
	byFolder := make(map[string][]indexEntry)
	for _, e := range entries {
		byFolder[e.Folder] = append(byFolder[e.Folder], e)
	}

	names := make([]string, 0, len(byFolder))
	for name := range byFolder {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if (names[i] == unfiledName) != (names[j] == unfiledName) {
			return names[j] == unfiledName
		}
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	data := indexData{Title: indexTitle}
	for _, name := range names {
		data.Folders = append(data.Folders, indexFolder{Name: name, Conversations: byFolder[name]})
	}

	var buf bytes.Buffer
	if err := h.index.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render index page: %w", err)
	}
	return buf.Bytes(), nil
}

// imageExtensions are attachment extensions shown inline as images.
var imageExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".bmp": {}, ".svg": {},
}

// isImageName reports whether name has an image extension.
func isImageName(name string) bool {
	_, ok := imageExtensions[strings.ToLower(filepath.Ext(name))]
	return ok
}
