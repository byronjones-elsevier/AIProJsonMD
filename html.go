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
	goldhtml "github.com/yuin/goldmark/renderer/html"
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
img{max-width:100%}
.breadcrumb ol{display:flex;flex-wrap:wrap;gap:.25rem;margin:0 0 1rem;padding:0;list-style:none;color:var(--muted)}
.breadcrumb li+li::before{content:"\203A";margin-right:.25rem}
.breadcrumb li[aria-current]{color:var(--fg);overflow-wrap:anywhere}
.code-block{position:relative}
.copy-button{position:absolute;top:.4rem;right:.4rem;padding:.15rem .5rem;font:inherit;font-size:.8rem;color:var(--fg);background:var(--bg);border:1px solid var(--border);border-radius:4px;cursor:pointer;opacity:.75}
.copy-button:hover,.copy-button:focus-visible{opacity:1}
.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0);white-space:nowrap}
@media print{.copy-button{display:none}}`

// pageHead is the shared document head. The Content-Security-Policy (see
// contentSecurityPolicy) blocks every external resource and allows no script
// except the copy-button script, by hash, on pages with a code block, so a
// hostile chat cannot run code or phone home even if an escaping bug were
// introduced.
const pageHead = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="{{.CSP}}">
<title>{{.Title}}</title>
<style>` + pageStyle + `</style>
</head>
<body>
<main>
`

// navBlock is the breadcrumb trail shown on every page below the main index.
// Its links are the navigation: the first crumb is the main index, the one
// before the current page is the parent, and the current page is the last crumb
// with no link.
const navBlock = `<nav class="breadcrumb" aria-label="Breadcrumb">
<ol>
{{range .Nav.Crumbs}}<li{{if not .Href}} aria-current="page"{{end}}>{{if .Href}}<a href="{{.Href}}">{{.Label}}</a>{{else}}{{.Label}}{{end}}</li>
{{end}}</ol>
</nav>
`

const pageTemplate = pageHead + navBlock + `<h1>{{.Title}}</h1>
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
{{if .Script}}<script>{{.Script}}</script>
{{end}}</body>
</html>
`

const folderTemplate = pageHead + navBlock + `<h1>{{.Title}}</h1>
<ul>
{{range .Conversations}}<li><a href="{{.Href}}">{{.Title}}</a></li>
{{end}}</ul>
</main>
</body>
</html>
`

const indexTemplate = pageHead + `<h1>{{.Title}}</h1>
{{range .Folders}}<h2><a href="{{.Href}}">{{.Name}}</a></h2>
<ul>
{{range .Conversations}}<li><a href="{{.Href}}">{{.Title}}</a></li>
{{end}}</ul>
{{end}}</main>
</body>
</html>
`

// crumb is one step in the breadcrumb trail. Href is empty for the current page.
type crumb struct {
	Label string
	Href  string
}

// navData is the navigation shown on pages below the main index: a breadcrumb
// trail from the main index down to the current page.
type navData struct {
	Crumbs []crumb
}

// pageData is the model for one conversation page.
type pageData struct {
	Title       string
	ID          string
	CSP         string
	Script      template.JS
	Nav         navData
	Messages    []messageData
	Attachments []attachmentData
}

// folderLink is one conversation listed on a folder page, linked relative to
// the folder.
type folderLink struct {
	Title string
	Href  string
}

// folderPageData is the model for a folder's index.html.
type folderPageData struct {
	Title         string
	CSP           string
	Nav           navData
	Conversations []folderLink
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

// indexEntry is one conversation written as HTML. Folder is the folder's
// directory name, Href is the escaped path from the output directory, and File
// is the unescaped HTML file name inside the folder.
type indexEntry struct {
	Folder string
	Title  string
	Href   string
	File   string
}

// indexFolder groups index entries under a folder name; Href is the folder's
// own page.
type indexFolder struct {
	Name          string
	Href          string
	Conversations []indexEntry
}

// indexData is the model for index.html.
type indexData struct {
	Title   string
	CSP     string
	Folders []indexFolder
}

// htmlRenderer renders conversations and the index page as HTML.
type htmlRenderer struct {
	markdown goldmark.Markdown
	page     *template.Template
	folder   *template.Template
	index    *template.Template
}

// newHTMLRenderer parses the page templates and builds a Markdown renderer
// that escapes, rather than passes through or drops, raw HTML found in chats.
func newHTMLRenderer() (*htmlRenderer, error) {
	page, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse page template: %w", err)
	}
	folder, err := template.New("folder").Parse(folderTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse folder template: %w", err)
	}
	index, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse index template: %w", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.TaskList),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(util.Prioritized(escapeRawHTML{}, 100)),
			// A single newline in a chat message is a line break the author typed;
			// plain Markdown would join the lines with a space.
			goldhtml.WithHardWraps(),
		),
	)
	return &htmlRenderer{markdown: md, page: page, folder: folder, index: index}, nil
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

// hasCodeBlock reports whether rendered message HTML contains a code block,
// which is what the copy-button script acts on.
func hasCodeBlock(body template.HTML) bool {
	return strings.Contains(string(body), "<pre><code")
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

// folderIndexName is the page listing a folder's conversations. It sits in the
// folder next to the conversation pages, so a conversation links up with it.
const folderIndexName = reservedConversationName + ".html"

// conversationNav is the breadcrumb for a conversation page: Main index >
// folder > conversation. homeHref is the escaped path from the conversation
// page to the main index.
func conversationNav(folder, title, homeHref string) navData {
	return navData{Crumbs: []crumb{
		{Label: "Main index", Href: homeHref},
		{Label: folder, Href: folderIndexName},
		{Label: title},
	}}
}

// folderNav is the breadcrumb for a folder page: Main index > folder. homeHref
// is the escaped path from the folder page to the main index.
func folderNav(folder, homeHref string) navData {
	return navData{Crumbs: []crumb{{Label: "Main index", Href: homeHref}, {Label: folder}}}
}

// renderFolder renders a folder's index.html listing its conversations. Entry
// links are relative to the folder.
func (h *htmlRenderer) renderFolder(folder, homeHref string, entries []indexEntry) ([]byte, error) {
	data := folderPageData{Title: folder, CSP: baseCSP, Nav: folderNav(folder, homeHref)}
	for _, e := range entries {
		data.Conversations = append(data.Conversations, folderLink{Title: e.Title, Href: escapeLinkPath(e.File)})
	}

	var buf bytes.Buffer
	if err := h.folder.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render folder page: %w", err)
	}
	return buf.Bytes(), nil
}

// renderConversation renders one conversation page with its navigation.
func (h *htmlRenderer) renderConversation(title, id string, nav navData, messages []Message, attachments []attachment) ([]byte, error) {
	data := pageData{Title: title, ID: id, Nav: nav}

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
		if hasCodeBlock(body) {
			data.Script = template.JS(copyScript) //nolint:gosec // a constant we control; the page policy allows exactly this script by hash
		}
	}
	data.CSP = contentSecurityPolicy(string(data.Script))

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

	data := indexData{Title: indexTitle, CSP: baseCSP}
	for _, name := range names {
		data.Folders = append(data.Folders, indexFolder{
			Name:          name,
			Href:          escapeLinkPath(name + "/" + folderIndexName),
			Conversations: byFolder[name],
		})
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
