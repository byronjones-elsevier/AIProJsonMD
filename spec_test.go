package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// readSpec returns the text of ENGINEERING.md, the primary specification.
func readSpec(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("ENGINEERING.md")
	if err != nil {
		t.Fatalf("read ENGINEERING.md: %v", err)
	}
	return string(data)
}

// TestSpecMatchesCode fails when ENGINEERING.md has drifted from the code in the
// places where it quotes the code literally. Update both in the same change.
func TestSpecMatchesCode(t *testing.T) {
	spec := readSpec(t)

	literals := map[string]string{
		"Appendix A: stylesheet (pageStyle)":                  pageStyle,
		"Appendix B: head (pageHead with {{pageStyle}})":      strings.Replace(pageHead, pageStyle, "{{pageStyle}}", 1),
		"Appendix B: breadcrumb block (navBlock)":             navBlock,
		"Appendix B: conversation page tail (pageTemplate)":   strings.TrimPrefix(pageTemplate, pageHead+navBlock),
		"Appendix B: folder page tail (folderTemplate)":       strings.TrimPrefix(folderTemplate, pageHead+navBlock),
		"Appendix B: main index tail (indexTemplate)":         strings.TrimPrefix(indexTemplate, pageHead),
		"Appendix C: copy script (copyScript)":                copyScript,
		"Section 5.9: base Content-Security-Policy (baseCSP)": baseCSP,
	}
	for name, want := range literals {
		if !strings.Contains(spec, want) {
			t.Errorf("ENGINEERING.md does not contain the current %s", name)
		}
	}
}

// TestSpecStatesToolchainVersions checks that section 9 names the versions pinned in go.mod.
func TestSpecStatesToolchainVersions(t *testing.T) {
	spec := readSpec(t)
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	if !strings.Contains(spec, strings.TrimSpace(string(mod))) {
		t.Errorf("ENGINEERING.md section 9 does not contain the current go.mod verbatim")
	}

	patterns := map[string]*regexp.Regexp{
		"go directive":        regexp.MustCompile(`(?m)^go (\S+)$`),
		"toolchain directive": regexp.MustCompile(`(?m)^toolchain (\S+)$`),
		"goldmark version":    regexp.MustCompile(`(?m)^require github\.com/yuin/goldmark (\S+)$`),
	}
	want := map[string]string{
		"go directive":        "go %s",
		"toolchain directive": "toolchain %s",
		"goldmark version":    "github.com/yuin/goldmark %s",
	}
	for name, re := range patterns {
		m := re.FindSubmatch(mod)
		if m == nil {
			t.Errorf("go.mod has no %s", name)
			continue
		}
		token := strings.Replace(want[name], "%s", string(m[1]), 1)
		if !strings.Contains(spec, token) {
			t.Errorf("ENGINEERING.md section 9 does not state %q (from go.mod)", token)
		}
	}
}

// TestSpecWindowsReservedNamesMatchCode checks that the set described in
// section 5.3 (CON, PRN, AUX, NUL, COM1 to COM9, LPT1 to LPT9) is the set in the code.
func TestSpecWindowsReservedNamesMatchCode(t *testing.T) {
	spec := readSpec(t)
	for _, phrase := range []string{"`CON`, `PRN`, `AUX`, `NUL`", "`COM1` to `COM9`", "`LPT1` to `LPT9`"} {
		if !strings.Contains(spec, phrase) {
			t.Errorf("ENGINEERING.md section 5.3 does not contain %q", phrase)
		}
	}

	want := map[string]struct{}{"CON": {}, "PRN": {}, "AUX": {}, "NUL": {}}
	for _, prefix := range []string{"COM", "LPT"} {
		for n := '1'; n <= '9'; n++ {
			want[prefix+string(n)] = struct{}{}
		}
	}
	if len(windowsReservedNames) != len(want) {
		t.Errorf("code reserves %d names, specification describes %d", len(windowsReservedNames), len(want))
	}
	for name := range want {
		if _, ok := windowsReservedNames[name]; !ok {
			t.Errorf("specification reserves %q but the code does not", name)
		}
	}
}

// TestSpecVerificationTableNamesRealTests checks that every test named in
// section 11.1 exists. A name ending in "*" matches any test with that prefix.
func TestSpecVerificationTableNamesRealTests(t *testing.T) {
	spec := readSpec(t)

	start := strings.Index(spec, "### 11.1 Acceptance tests")
	end := strings.Index(spec, "### 11.2 ")
	if start < 0 || end < start {
		t.Fatalf("could not find section 11.1 in ENGINEERING.md")
	}
	table := spec[start:end]

	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	declared := regexp.MustCompile(`(?m)^func (Test\w+)\(`)
	exists := map[string]struct{}{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, m := range declared.FindAllStringSubmatch(string(data), -1) {
			exists[m[1]] = struct{}{}
		}
	}

	named := regexp.MustCompile("`(Test[A-Za-z0-9_]*\\*?)`")
	var missing []string
	for _, m := range named.FindAllStringSubmatch(table, -1) {
		name := m[1]
		if prefix, wildcard := strings.CutSuffix(name, "*"); wildcard {
			found := false
			for e := range exists {
				if strings.HasPrefix(e, prefix) {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, name)
			}
			continue
		}
		if _, ok := exists[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("section 11.1 names tests that do not exist: %v", missing)
	}
}

// fencedBlock is one fenced code block found in the specification.
type fencedBlock struct {
	lang string
	body string
}

// fencedBlocks returns the fenced code blocks that lie between the markers from
// and to in the specification. A fence is three or more backticks and is closed
// by a line of exactly the same fence.
func fencedBlocks(t *testing.T, spec, from, to string) []fencedBlock {
	t.Helper()
	start := strings.Index(spec, from)
	if start < 0 {
		t.Fatalf("marker %q not found in ENGINEERING.md", from)
	}
	rest := spec[start+len(from):]
	if end := strings.Index(rest, to); end >= 0 {
		rest = rest[:end]
	}

	var blocks []fencedBlock
	lines := strings.Split(rest, "\n")
	for i := 0; i < len(lines); i++ {
		n := len(lines[i]) - len(strings.TrimLeft(lines[i], "`"))
		if n < 3 {
			continue
		}
		lang := strings.TrimSpace(lines[i][n:])
		fence := strings.Repeat("`", n)
		var body []string
		for i++; i < len(lines) && lines[i] != fence; i++ {
			body = append(body, lines[i])
		}
		blocks = append(blocks, fencedBlock{lang: lang, body: strings.Join(body, "\n")})
	}
	return blocks
}

// TestSpecWorkedExample runs the program on the input in section 4.5 of the
// specification and checks the summary, the list of files, the Markdown file,
// and the HTML fragments against the specification's own expected output.
func TestSpecWorkedExample(t *testing.T) {
	spec := readSpec(t)
	blocks := fencedBlocks(t, spec, "### 4.5 Worked example", "## 5. Behavior and algorithms")
	pick := func(lang, prefix string) string {
		t.Helper()
		for _, b := range blocks {
			if b.lang == lang && strings.HasPrefix(b.body, prefix) {
				return b.body
			}
		}
		t.Fatalf("section 4.5 has no %s block starting with %q", lang, prefix)
		return ""
	}

	input := pick("json", "{")
	command := pick("text", "$ AIProJsonMD")
	wantFiles := pick("text", "out/index.html")
	wantMarkdown := pick("markdown", "# Deploy: notes")
	wantConversation := pick("html", `<nav class="breadcrumb"`)
	wantCSP := pick("text", "default-src")
	wantFolder := pick("html", "<h1>Project A</h1>")
	wantIndex := pick("html", "<h1>AI Pro Export</h1>")

	t.Chdir(t.TempDir())
	if err := os.WriteFile("export.json", []byte(input), 0o600); err != nil {
		t.Fatalf("write export: %v", err)
	}
	var stdout bytes.Buffer
	if err := run([]string{"-in", "export.json", "-out", "out", "-format", "both"}, &stdout); err != nil {
		t.Fatalf("run: %v", err)
	}

	// The summary is the last three lines of the command block (the first two
	// are the command and the warning, which goes to standard error).
	commandLines := strings.Split(command, "\n")
	wantSummary := strings.Join(commandLines[len(commandLines)-3:], "\n") + "\n"
	if stdout.String() != wantSummary {
		t.Errorf("summary =\n%s\nwant\n%s", stdout.String(), wantSummary)
	}

	var gotFiles []string
	err := filepath.WalkDir("out", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			gotFiles = append(gotFiles, filepath.ToSlash(p))
		}
		return err
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	expectedFiles := strings.Split(wantFiles, "\n")
	sort.Strings(gotFiles)
	sort.Strings(expectedFiles)
	if !slices.Equal(gotFiles, expectedFiles) {
		t.Errorf("files written:\n%v\nwant\n%v", gotFiles, expectedFiles)
	}

	if got := readFile(t, "out/Project A/Deploy_ notes.md"); got != wantMarkdown+"\n" {
		t.Errorf("Markdown file =\n%s\nwant\n%s", got, wantMarkdown+"\n")
	}

	conversation := readFile(t, "out/Project A/Deploy_ notes.html")
	if !strings.Contains(conversation, wantConversation) {
		t.Errorf("conversation page does not contain the body shown in section 4.5")
	}
	if got := cspOf(t, conversation); got != wantCSP {
		t.Errorf("policy = %q, want %q", got, wantCSP)
	}
	if !strings.Contains(readFile(t, "out/Project A/index.html"), wantFolder) {
		t.Errorf("folder page does not contain the body shown in section 4.5")
	}
	if !strings.Contains(readFile(t, "out/index.html"), wantIndex) {
		t.Errorf("main index does not contain the body shown in section 4.5")
	}
}

// TestSpecRenderingExamples runs every input and output pair under "Rendering
// examples" in section 5.8 through the Markdown renderer.
func TestSpecRenderingExamples(t *testing.T) {
	spec := readSpec(t)
	blocks := fencedBlocks(t, spec, "#### Rendering examples", "### 5.9 ")
	if len(blocks) < 2 || len(blocks)%2 != 0 {
		t.Fatalf("expected input and output block pairs, found %d blocks", len(blocks))
	}

	r := newTestRenderer(t)
	for i := 0; i < len(blocks); i += 2 {
		in, out := blocks[i], blocks[i+1]
		if in.lang != "markdown" || out.lang != "html" {
			t.Fatalf("pair %d is %s then %s, want markdown then html", i/2+1, in.lang, out.lang)
		}
		got, err := r.markdownToHTML(in.body)
		if err != nil {
			t.Fatalf("pair %d: %v", i/2+1, err)
		}
		if string(got) != out.body+"\n" {
			t.Errorf("pair %d: input %q\n got  %q\n want %q", i/2+1, in.body, string(got), out.body+"\n")
		}
	}
}

// TestSpecEveryRequirementIsVerified checks REQ-NFR-5: every requirement defined
// in section 2 is mentioned in section 11, where its tests or procedure are named.
func TestSpecEveryRequirementIsVerified(t *testing.T) {
	spec := readSpec(t)
	between := func(from, to string) string {
		i, j := strings.Index(spec, from), strings.Index(spec, to)
		if i < 0 || j < i {
			t.Fatalf("cannot find %q and %q in ENGINEERING.md", from, to)
		}
		return spec[i:j]
	}
	requirements := between("## 2. Requirements", "## 3. External interfaces")
	verification := between("## 11. Verification plan", "## 12. Decisions")

	defined := regexp.MustCompile(`(?m)^- (REQ-[A-Z]+-\d+):`).FindAllStringSubmatch(requirements, -1)
	if len(defined) == 0 {
		t.Fatalf("found no requirements in section 2")
	}

	// A reference is REQ-AREA-N, optionally followed by " to M" or ", M, K".
	reference := regexp.MustCompile(`REQ-([A-Z]+)-(\d+)((?:\s+to\s+\d+)?)((?:,\s*\d+)*)`)
	digits := regexp.MustCompile(`\d+`)
	covered := map[string]bool{}
	for _, m := range reference.FindAllStringSubmatch(verification, -1) {
		prefix := "REQ-" + m[1] + "-"
		covered[prefix+m[2]] = true
		if m[3] != "" {
			first, _ := strconv.Atoi(m[2])
			last, _ := strconv.Atoi(digits.FindString(m[3]))
			for n := first; n <= last; n++ {
				covered[prefix+strconv.Itoa(n)] = true
			}
		}
		for _, extra := range digits.FindAllString(m[4], -1) {
			covered[prefix+extra] = true
		}
	}

	var missing []string
	for _, d := range defined {
		if !covered[d[1]] {
			missing = append(missing, d[1])
		}
	}
	if len(missing) > 0 {
		t.Errorf("section 11 does not mention these requirements: %v", missing)
	}
}

// TestOnlyGoldmarkIsRequired checks REQ-NFR-2: go.mod requires exactly one
// module, goldmark.
func TestOnlyGoldmarkIsRequired(t *testing.T) {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	required := regexp.MustCompile(`(?m)^(?:require\s+)?([\w.\-]+(?:/[\w.\-]+)+)\s+v[\w.\-+]+`).FindAllSubmatch(mod, -1)
	var names []string
	for _, m := range required {
		names = append(names, string(m[1]))
	}
	if len(names) != 1 || names[0] != "github.com/yuin/goldmark" {
		t.Errorf("go.mod requires %v, want only github.com/yuin/goldmark", names)
	}
}
