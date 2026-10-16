package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// requireTools skips the test when bash or git is not installed.
func requireTools(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
}

// git runs a git command in dir with fixed identity and the given committer and
// author date, failing the test on error.
func git(t *testing.T, dir, date string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", full...) //nolint:gosec // test helper; the arguments are fixed strings from this file
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// mergeBranch creates a branch from main with one empty commit per date, then
// merges it into main with a merge commit made at mergeDate.
func mergeBranch(t *testing.T, dir, branch, subject, mergeDate string, commitDates ...string) {
	t.Helper()
	git(t, dir, commitDates[0], "checkout", "-q", "-b", branch, "main")
	for i, date := range commitDates {
		git(t, dir, date, "commit", "-q", "--allow-empty", "-m", branch+" commit "+string(rune('a'+i)))
	}
	git(t, dir, mergeDate, "checkout", "-q", "main")
	git(t, dir, mergeDate, "merge", "-q", "--no-ff", "-m", subject, branch)
}

// runChangelogStatus runs scripts/changelog-status.sh in dir, with a time zone set
// so that a script that forgot to convert to UTC would show the wrong times.
func runChangelogStatus(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("scripts", "changelog-status.sh"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	cmd := exec.Command("bash", append([]string{script}, args...)...) //nolint:gosec // test helper; the script path is this repository's own script
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TZ=America/New_York")
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut

	runErr := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case runErr == nil:
		exitCode = 0
	case errors.As(runErr, &exitErr):
		exitCode = exitErr.ExitCode()
	default:
		t.Fatalf("run script: %v", runErr)
	}
	return out.String(), errOut.String(), exitCode
}

// TestChangelogStatusScript checks that the script reads pull request numbers,
// merge times, and commit counts and times from merge commits, in UTC.
func TestChangelogStatusScript(t *testing.T) {
	requireTools(t)

	dir := t.TempDir()
	git(t, dir, "2025-12-31T00:00:00+00:00", "init", "-q", "-b", "main")
	git(t, dir, "2025-12-31T00:00:00+00:00", "commit", "-q", "--allow-empty", "-m", "init")

	// Two commits with different offsets, merged at a time with another offset.
	mergeBranch(t, dir, "feat/x", "Merge pull request #7 from octo/feat/x", "2026-01-02T05:04:05-02:00",
		"2026-01-01T10:00:00+00:00", "2026-01-01T11:30:00-02:00")
	// A branch name with a slash in a second path component.
	mergeBranch(t, dir, "docs/y", "Merge pull request #8 from octo/docs/y", "2026-01-04T12:00:00+00:00",
		"2026-01-03T12:00:00+00:00")
	// The same branch name used again for a later pull request.
	git(t, dir, "2026-01-05T00:00:00+00:00", "branch", "-q", "-D", "feat/x")
	mergeBranch(t, dir, "feat/x", "Merge pull request #9 from octo/feat/x", "2026-01-06T00:00:00+00:00",
		"2026-01-05T08:00:00+05:30")
	// A merge that does not follow the pull request subject is ignored.
	mergeBranch(t, dir, "other", "Merge branch 'other'", "2026-01-07T00:00:00+00:00",
		"2026-01-06T12:00:00+00:00")

	changelog := "# Changelog\n\n" +
		"### feat/x: first thing\n- Branch: `feat/x`\n\n" +
		"### docs/y: second thing\n\n" +
		"### unmerged/z: never merged\n\n" +
		"### other: merged without a PR number\n\n" +
		"### Initial development (direct commits to main)\n"
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(changelog), 0o600); err != nil {
		t.Fatalf("write changelog: %v", err)
	}

	stdout, stderr, code := runChangelogStatus(t, dir)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}

	want := []string{
		"branch\tstatus\tpr\tmerged_utc\tcommits\tfirst_commit_utc\tlast_commit_utc",
		"feat/x\tmerged\t#9\t2026-01-06T00:00:00Z\t1\t2026-01-05T02:30:00Z\t2026-01-05T02:30:00Z",
		"feat/x\tmerged\t#7\t2026-01-02T07:04:05Z\t2\t2026-01-01T10:00:00Z\t2026-01-01T13:30:00Z",
		"docs/y\tmerged\t#8\t2026-01-04T12:00:00Z\t1\t2026-01-03T12:00:00Z\t2026-01-03T12:00:00Z",
		"unmerged/z\tnot merged\t-\t-\t-\t-\t-",
		"other\tnot merged\t-\t-\t-\t-\t-",
		"Initial development (direct commits to main)\tnot merged\t-\t-\t-\t-\t-",
	}
	got := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("output:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// The table option aligns the same rows.
	table, _, code := runChangelogStatus(t, dir, "-t")
	if code != 0 || !strings.Contains(table, "docs/y") || strings.Contains(table, "\t") {
		t.Errorf("-t output should be aligned without tabs (exit %d):\n%s", code, table)
	}
}

// TestChangelogStatusScriptErrors checks the help text and the exit statuses.
func TestChangelogStatusScriptErrors(t *testing.T) {
	requireTools(t)

	dir := t.TempDir()
	git(t, dir, "2026-01-01T00:00:00+00:00", "init", "-q", "-b", "main")
	git(t, dir, "2026-01-01T00:00:00+00:00", "commit", "-q", "--allow-empty", "-m", "init")
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n"), 0o600); err != nil {
		t.Fatalf("write changelog: %v", err)
	}

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{"help", []string{"-h"}, 0, "Usage: changelog-status.sh", ""},
		{"unknown option", []string{"-z"}, 2, "", "unknown option -z"},
		{"option without value", []string{"-r"}, 2, "", "option -r needs an argument"},
		{"missing file", []string{"-f", "nope.md"}, 1, "", "changelog file not found: nope.md"},
		{"unknown ref", []string{"-r", "no-such-ref"}, 1, "", "unknown git ref: no-such-ref"},
		{"empty changelog prints only the header", nil, 0, "branch\tstatus\tpr", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := runChangelogStatus(t, dir, tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit = %d, want %d (stderr: %s)", code, tt.wantCode, stderr)
			}
			if !strings.Contains(stdout, tt.wantStdout) {
				t.Errorf("stdout %q does not contain %q", stdout, tt.wantStdout)
			}
			if !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr, tt.wantStderr)
			}
		})
	}

	outside := t.TempDir()
	if _, stderr, code := runChangelogStatus(t, outside); code != 1 || !strings.Contains(stderr, "not inside a git repository") {
		t.Errorf("outside a repository: exit %d, stderr %q", code, stderr)
	}
}

// TestChangelogEntriesHaveNoPostMergeFacts enforces REQ-PROC-1: entries written
// under "Entries" in CHANGELOG.md hold no facts that exist only after a pull
// request is opened or merged (they come from `make changelog/status`).
func TestChangelogEntriesHaveNoPostMergeFacts(t *testing.T) {
	data, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	text := string(data)

	start := strings.Index(text, "\n## Entries\n")
	end := strings.Index(text, "\n## Backfilled history\n")
	if start < 0 || end < start {
		t.Fatalf("CHANGELOG.md needs an \"Entries\" section followed by a \"Backfilled history\" section")
	}
	entries := text[start:end]
	if !strings.Contains(entries, "\n### ") {
		t.Errorf("the Entries section has no entries")
	}

	forbidden := map[string]*regexp.Regexp{
		"a pull request number":    regexp.MustCompile(`(?i)\bPR #\d+|pull request #\d+`),
		"a Merged line":            regexp.MustCompile(`(?m)^- Merged:`),
		"a Status line":            regexp.MustCompile(`(?m)^- Status:`),
		"a Started line":           regexp.MustCompile(`(?m)^- Started:`),
		"a Last commit line":       regexp.MustCompile(`(?m)^- Last commit:`),
		"a commit id in backticks": regexp.MustCompile("`[0-9a-f]{7,40}(?:\\.\\.[0-9a-f]{7,40})?`"),
	}
	for what, re := range forbidden {
		if loc := re.FindStringIndex(entries); loc != nil {
			line := strings.Count(text[:start+loc[0]], "\n") + 1
			t.Errorf("CHANGELOG.md line %d contains %s, which must come from `make changelog/status`", line, what)
		}
	}
}
